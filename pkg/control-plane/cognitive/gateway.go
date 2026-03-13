package cognitive

import (
	"context"
)

// Gateway is the abstraction for AI services (navigation, recognition, planning, speech).
type Gateway interface {
	// Navigate computes a path from A to B for the given robot.
	Navigate(ctx context.Context, req NavigateRequest) (*NavigateResult, error)
	// Recognize detects objects in sensor data.
	Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error)
	// Plan generates a sequence of steps for a task type.
	Plan(ctx context.Context, req PlanRequest) (*PlanResult, error)
	// Transcribe converts speech audio to text (STT).
	Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResult, error)
	// Synthesize converts text to speech audio (TTS).
	Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResult, error)
	// UnderstandIntent extracts structured intent from user text.
	UnderstandIntent(ctx context.Context, req UnderstandIntentRequest) (*IntentResult, error)
	// Translate translates text to the target language.
	Translate(ctx context.Context, req TranslateRequest) (*TranslateResult, error)
}
