package speech

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/sai-aurosy/platform/pkg/control-plane/cognitive"
	"github.com/sai-aurosy/platform/pkg/control-plane/conversations"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

// Pipeline processes speech: Transcribe -> UnderstandIntent -> ConversationCatalog -> Synthesize -> Publish.
type Pipeline struct {
	gateway    cognitive.Gateway
	catalog    *conversations.Catalog
	bus        *telemetry.Bus
}

// NewPipeline creates a new speech pipeline.
func NewPipeline(gateway cognitive.Gateway, catalog *conversations.Catalog, bus *telemetry.Bus) *Pipeline {
	return &Pipeline{
		gateway: gateway,
		catalog: catalog,
		bus:     bus,
	}
}

// ProcessResult holds the result of a full speech pipeline run.
type ProcessResult struct {
	Transcript   string                 `json:"transcript"`
	Language     string                 `json:"language"`
	Intent       string                 `json:"intent"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Response     string                 `json:"response"`
	AudioBase64  string                 `json:"audio_base64"`
}

// Process runs the full pipeline: audio -> STT -> intent -> conversation -> TTS.
// Returns the result; caller may publish to NATS if needed.
func (p *Pipeline) Process(ctx context.Context, robotID, tenantID string, audioBase64 string) (*ProcessResult, error) {
	// 1. Transcribe
	transcribeReq := cognitive.TranscribeRequest{
		RobotID:     robotID,
		AudioBase64: audioBase64,
	}
	transcribeRes, err := p.gateway.Transcribe(ctx, transcribeReq)
	if err != nil {
		return nil, err
	}
	if transcribeRes.Text == "" {
		return &ProcessResult{
			Transcript:  "",
			Language:    transcribeRes.Language,
			Intent:      "",
			Parameters:  nil,
			Response:    "",
			AudioBase64: "",
		}, nil
	}

	// 2. Understand intent
	intentReq := cognitive.UnderstandIntentRequest{
		RobotID:  robotID,
		Text:     transcribeRes.Text,
		Language: transcribeRes.Language,
	}
	intentRes, err := p.gateway.UnderstandIntent(ctx, intentReq)
	if err != nil {
		return nil, err
	}

	// 3. Lookup conversation by intent
	conv, err := p.catalog.GetByIntent(ctx, intentRes.Intent, tenantID)
	if err != nil || conv == nil {
		return &ProcessResult{
			Transcript:  transcribeRes.Text,
			Language:    transcribeRes.Language,
			Intent:      intentRes.Intent,
			Parameters:  intentRes.Parameters,
			Response:    "",
			AudioBase64: "",
		}, nil
	}

	// 4. Resolve response from template
	params := make(map[string]interface{})
	for k, v := range intentRes.Parameters {
		params[k] = v
	}
	responseText := conversations.ResolveResponse(conv.ResponseTemplate, params, transcribeRes.Language)
	if responseText == "" {
		return &ProcessResult{
			Transcript:  transcribeRes.Text,
			Language:    transcribeRes.Language,
			Intent:      intentRes.Intent,
			Parameters:  intentRes.Parameters,
			Response:    "",
			AudioBase64: "",
		}, nil
	}

	// 4b. Translate to user language if not English
	lang := transcribeRes.Language
	if lang == "" {
		lang = "en"
	}
	if lang != "en" {
		translateRes, err := p.gateway.Translate(ctx, cognitive.TranslateRequest{
			RobotID:        robotID,
			Text:           responseText,
			TargetLanguage: lang,
		})
		if err == nil && translateRes.Text != "" {
			responseText = translateRes.Text
		}
	}

	// 5. Synthesize TTS
	synthReq := cognitive.SynthesizeRequest{
		RobotID:  robotID,
		Text:     responseText,
		Language: lang,
	}
	synthRes, err := p.gateway.Synthesize(ctx, synthReq)
	if err != nil {
		return nil, err
	}

	// 6. Publish to NATS (optional observability)
	if p.bus != nil {
		_ = p.publishObservability(robotID, transcribeRes, intentRes, responseText)
		if synthRes.AudioBase64 != "" {
			audioBytes, _ := base64.StdEncoding.DecodeString(synthRes.AudioBase64)
			_ = p.bus.PublishAudioOutput(robotID, audioBytes)
		}
	}

	return &ProcessResult{
		Transcript:  transcribeRes.Text,
		Language:    transcribeRes.Language,
		Intent:      intentRes.Intent,
		Parameters:  intentRes.Parameters,
		Response:    responseText,
		AudioBase64: synthRes.AudioBase64,
	}, nil
}

func (p *Pipeline) publishObservability(robotID string, transcript *cognitive.TranscribeResult, intent *cognitive.IntentResult, response string) error {
	if transcript != nil {
		data, _ := json.Marshal(transcript)
		_ = p.bus.PublishSpeechTranscript(robotID, data)
	}
	if intent != nil {
		data, _ := json.Marshal(intent)
		_ = p.bus.PublishSpeechIntent(robotID, data)
	}
	if response != "" {
		_ = p.bus.PublishSpeechResponse(robotID, []byte(response))
	}
	return nil
}
