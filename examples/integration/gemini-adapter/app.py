"""
Gemini Adapter — HTTP service implementing SAI AUROSY Cognitive Gateway contract.
Exposes /transcribe, /synthesize, /understand-intent using Google Gemini API.
"""
import base64
import io
import json
import os
import subprocess
import tempfile
import wave

from flask import Flask, request, jsonify
from google import genai
from google.genai import types
from pydub import AudioSegment

app = Flask(__name__)

API_KEY = os.environ.get("GEMINI_API_KEY") or os.environ.get("GOOGLE_API_KEY")
client = genai.Client(api_key=API_KEY) if API_KEY else None

STT_MODEL = "gemini-3-flash-preview"
TTS_MODEL = "gemini-2.5-flash-preview-tts"
INTENT_MODEL = "gemini-2.5-flash-lite"

# Map SAI AUROSY language codes to Gemini voice config (language hint)
LANG_TO_VOICE = {
    "en": "Kore",
    "ru": "Kore",
    "uz": "Kore",
    "az": "Kore",
    "ar": "Kore",
}


def _convert_webm_to_mp3(audio_b64: str) -> bytes:
    """Convert base64 webm/opus to mp3 bytes for Gemini (supports mp3, wav).
    Chrome MediaRecorder produces webm with opus; use subprocess ffmpeg for robustness.
    """
    raw = base64.b64decode(audio_b64)
    if len(raw) < 100:
        raise ValueError("Audio too short (min ~100 bytes); record at least 1 second")
    with tempfile.NamedTemporaryFile(suffix=".webm", delete=False) as f:
        f.write(raw)
        path = f.name
    try:
        # Try pydub first (simpler)
        try:
            seg = AudioSegment.from_file(path, format="webm")
        except Exception:
            # Fallback: ffmpeg via subprocess (handles fragmented webm better)
            out_path = path + ".mp3"
            try:
                subprocess.run(
                    ["ffmpeg", "-y", "-f", "webm", "-i", path, "-vn", "-acodec", "libmp3lame", "-f", "mp3", out_path],
                    check=True,
                    capture_output=True,
                    timeout=30,
                )
                with open(out_path, "rb") as fp:
                    data = fp.read()
                return data
            finally:
                if os.path.exists(out_path):
                    os.unlink(out_path)
        buf = io.BytesIO()
        seg.export(buf, format="mp3")
        return buf.getvalue()
    finally:
        os.unlink(path)


def _pcm_to_wav_base64(pcm_bytes: bytes, sample_rate: int = 24000) -> str:
    """Wrap PCM LINEAR16 in WAV header and return base64."""
    buf = io.BytesIO()
    with wave.open(buf, "wb") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(sample_rate)
        wf.writeframes(pcm_bytes)
    return base64.b64encode(buf.getvalue()).decode("ascii")


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok", "configured": bool(API_KEY)})


def _require_client():
    if not client:
        return jsonify({"error": "GEMINI_API_KEY or GOOGLE_API_KEY not set"}), 503
    return None


@app.route("/transcribe", methods=["POST"])
def transcribe():
    """SAI AUROSY contract: {robot_id, audio_base64, language?} -> {text, language, confidence}"""
    err = _require_client()
    if err:
        return err
    data = request.get_json() or {}
    audio_b64 = data.get("audio_base64", "")
    if not audio_b64:
        return jsonify({"text": "", "language": "en", "confidence": 0}), 200

    try:
        mp3_bytes = _convert_webm_to_mp3(audio_b64)
    except Exception as e:
        return jsonify({"error": str(e)}), 400

    prompt = """Transcribe the speech and detect its language. Return JSON only: {"text":"transcript here","language":"xx"} where language is one of: en, ru, uz, az, ar. No punctuation unless spoken."""
    try:
        try:
            response = client.models.generate_content(
                model=STT_MODEL,
                contents=[
                    prompt,
                    types.Part.from_bytes(data=mp3_bytes, mime_type="audio/mp3"),
                ],
                config=types.GenerateContentConfig(response_mime_type="application/json"),
            )
        except Exception:
            response = client.models.generate_content(
                model=STT_MODEL,
                contents=[
                    prompt,
                    types.Part.from_bytes(data=mp3_bytes, mime_type="audio/mp3"),
                ],
            )
        raw = (response.text or "").strip()
        text, lang = "", "en"
        if "{" in raw and "}" in raw:
            start, end = raw.index("{"), raw.rindex("}") + 1
            try:
                parsed = json.loads(raw[start:end])
                text = (parsed.get("text") or "").strip()
                lang = (parsed.get("language") or "en").lower()[:2]
                if lang not in ("en", "ru", "uz", "az", "ar"):
                    lang = "en"
            except json.JSONDecodeError:
                text = raw.strip()
        else:
            text = raw.strip()
        return jsonify({
            "text": text,
            "language": lang,
            "confidence": 0.9 if text else 0,
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/synthesize", methods=["POST"])
def synthesize():
    """SAI AUROSY contract: {robot_id, text, language} -> {audio_base64}"""
    err = _require_client()
    if err:
        return err
    data = request.get_json() or {}
    text = data.get("text", "").strip()
    lang = (data.get("language") or "en").lower()[:2]
    if not text:
        return jsonify({"audio_base64": ""}), 200

    voice = LANG_TO_VOICE.get(lang, "Kore")
    try:
        response = client.models.generate_content(
            model=TTS_MODEL,
            contents=text,
            config=types.GenerateContentConfig(
                response_modalities=["AUDIO"],
                speech_config=types.SpeechConfig(
                    voice_config=types.VoiceConfig(
                        prebuilt_voice_config=types.PrebuiltVoiceConfig(
                            voice_name=voice,
                        )
                    )
                ),
            ),
        )
    except Exception as e:
        return jsonify({"error": str(e)}), 500

    if not response.candidates or not response.candidates[0].content.parts:
        return jsonify({"audio_base64": ""}), 200

    part = response.candidates[0].content.parts[0]
    inline = getattr(part, "inline_data", None) or getattr(part, "inlineData", None)
    if not inline:
        return jsonify({"audio_base64": ""}), 200

    pcm = getattr(inline, "data", None) or b""
    if not pcm:
        return jsonify({"audio_base64": ""}), 200
    wav_b64 = _pcm_to_wav_base64(pcm)
    return jsonify({"audio_base64": wav_b64})


@app.route("/translate", methods=["POST"])
def translate():
    """SAI AUROSY contract: {robot_id, text, target_language} -> {text}"""
    err = _require_client()
    if err:
        return err
    data = request.get_json() or {}
    text = (data.get("text") or "").strip()
    target = (data.get("target_language") or "en").lower()[:2]
    if not text or target == "en":
        return jsonify({"text": text}), 200
    lang_names = {"ru": "Russian", "uz": "Uzbek", "az": "Azerbaijani", "ar": "Arabic"}
    target_name = lang_names.get(target, target)
    prompt = f"Translate the following text to {target_name}. Return only the translated text, nothing else. Text: {text}"
    try:
        response = client.models.generate_content(
            model=INTENT_MODEL,
            contents=prompt,
        )
        out = (response.text or "").strip()
        return jsonify({"text": out or text}), 200
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/understand-intent", methods=["POST"])
def understand_intent():
    """SAI AUROSY contract: {robot_id, text, language?, context?} -> {intent, parameters, confidence}"""
    err = _require_client()
    if err:
        return err
    data = request.get_json() or {}
    text = (data.get("text") or "").strip()
    if not text:
        return jsonify({"intent": "", "parameters": {}, "confidence": 0}), 200

    prompt = """Extract intent from mall visitor text. Return JSON only, no other text.
Format: {"intent":"find_store|greeting|goodbye|unknown","parameters":{"store_name":"..."},"confidence":0.0-1.0}
Supported intents: find_store (store_name), greeting, goodbye. For unknown or unclear text use intent "unknown".
Text: """
    try:
        response = client.models.generate_content(
            model=INTENT_MODEL,
            contents=prompt + text,
            config=types.GenerateContentConfig(
                response_mime_type="application/json",
            ),
        )
        raw = (response.text or "").strip()
        # Extract JSON block if wrapped in markdown
        if "{" in raw:
            start = raw.index("{")
            end = raw.rindex("}") + 1
            raw = raw[start:end]
        parsed = json.loads(raw)
        intent = parsed.get("intent", "unknown")
        params = parsed.get("parameters") or {}
        conf = float(parsed.get("confidence", 0.8))
        return jsonify({
            "intent": intent,
            "parameters": params,
            "confidence": conf,
        })
    except Exception as e:
        return jsonify({"intent": "unknown", "parameters": {}, "confidence": 0}), 200


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8001"))
    app.run(host="0.0.0.0", port=port)
