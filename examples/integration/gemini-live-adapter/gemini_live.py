"""
GeminiLive — handles interaction with the Gemini Live API.
Adapted from google-gemini/gemini-live-api-examples.
"""
import asyncio
import inspect
import logging

from google import genai
from google.genai import types

logger = logging.getLogger(__name__)

MALL_ASSISTANT_INSTRUCTION = """You are a helpful mall assistant robot. You greet visitors, help them find stores, and answer questions about the mall. Keep responses concise and friendly. Support multiple languages: English, Russian, Uzbek, Azerbaijani, Arabic. Respond in the same language the visitor uses."""


class GeminiLive:
    """Handles the interaction with the Gemini Live API."""

    def __init__(self, api_key, model, input_sample_rate, tools=None, tool_mapping=None):
        self.api_key = api_key
        self.model = model
        self.input_sample_rate = input_sample_rate
        self.client = genai.Client(api_key=api_key, http_options={"api_version": "v1alpha"})
        self.tools = tools or []
        self.tool_mapping = tool_mapping or {}

    async def start_session(
        self,
        audio_input_queue,
        video_input_queue,
        text_input_queue,
        audio_output_callback,
        audio_interrupt_callback=None,
    ):
        config = types.LiveConnectConfig(
            response_modalities=[types.Modality.AUDIO],
            speech_config=types.SpeechConfig(
                voice_config=types.VoiceConfig(
                    prebuilt_voice_config=types.PrebuiltVoiceConfig(voice_name="Iapetus")
                )
            ),
            system_instruction=types.Content(parts=[types.Part(text=MALL_ASSISTANT_INSTRUCTION)]),
            input_audio_transcription=types.AudioTranscriptionConfig(),
            output_audio_transcription=types.AudioTranscriptionConfig(),
            proactivity=types.ProactivityConfig(proactive_audio=False),
            enable_affective_dialog=True,
            thinking_config=types.ThinkingConfig(thinking_budget=0),
            realtime_input_config=types.RealtimeInputConfig(
                automatic_activity_detection=types.AutomaticActivityDetection(
                    silence_duration_ms=400,
                    end_of_speech_sensitivity=types.EndSensitivity.END_SENSITIVITY_HIGH,
                )
            ),
            tools=self.tools,
        )

        async with self.client.aio.live.connect(model=self.model, config=config) as session:
            async def send_audio():
                try:
                    while True:
                        chunk = await audio_input_queue.get()
                        await session.send_realtime_input(
                            audio=types.Blob(
                                data=chunk,
                                mime_type=f"audio/pcm;rate={self.input_sample_rate}",
                            )
                        )
                except asyncio.CancelledError:
                    pass

            async def send_video():
                try:
                    while True:
                        chunk = await video_input_queue.get()
                        logger.info("Sending video frame to Gemini: %d bytes", len(chunk))
                        await session.send_realtime_input(
                            video=types.Blob(data=chunk, mime_type="image/jpeg")
                        )
                except asyncio.CancelledError:
                    pass

            async def send_text():
                try:
                    while True:
                        text = await text_input_queue.get()
                        logger.info("Sending text to Gemini: %s", text)
                        await session.send_realtime_input(text=text)
                except asyncio.CancelledError:
                    pass

            event_queue = asyncio.Queue()

            async def receive_loop():
                try:
                    async for response in session.receive():
                        server_content = response.server_content
                        tool_call = response.tool_call

                        if server_content:
                            if server_content.model_turn:
                                for part in server_content.model_turn.parts:
                                    inline = getattr(part, "inline_data", None) or getattr(
                                        part, "inlineData", None
                                    )
                                    if inline and getattr(inline, "data", None):
                                        data = inline.data
                                        if inspect.iscoroutinefunction(audio_output_callback):
                                            await audio_output_callback(data)
                                        else:
                                            audio_output_callback(data)

                            if server_content.input_transcription and server_content.input_transcription.text:
                                await event_queue.put(
                                    {"type": "user", "text": server_content.input_transcription.text}
                                )

                            if server_content.output_transcription and server_content.output_transcription.text:
                                await event_queue.put(
                                    {
                                        "type": "gemini",
                                        "text": server_content.output_transcription.text,
                                    }
                                )

                            if server_content.turn_complete:
                                await event_queue.put({"type": "turn_complete"})

                            if server_content.interrupted:
                                if audio_interrupt_callback:
                                    if inspect.iscoroutinefunction(audio_interrupt_callback):
                                        await audio_interrupt_callback()
                                    else:
                                        audio_interrupt_callback()
                                await event_queue.put({"type": "interrupted"})

                        if tool_call:
                            function_responses = []
                            for fc in tool_call.function_calls:
                                func_name = fc.name
                                args = fc.args or {}
                                result = f"Error: tool not implemented"

                                if func_name in self.tool_mapping:
                                    try:
                                        tool_func = self.tool_mapping[func_name]
                                        if inspect.iscoroutinefunction(tool_func):
                                            result = await tool_func(**args)
                                        else:
                                            loop = asyncio.get_running_loop()
                                            result = await loop.run_in_executor(
                                                None, lambda: tool_func(**args)
                                            )
                                    except Exception as e:
                                        result = f"Error: {e}"

                                function_responses.append(
                                    types.FunctionResponse(
                                        name=func_name,
                                        id=fc.id,
                                        response={"result": result},
                                    )
                                )
                                await event_queue.put(
                                    {"type": "tool_call", "name": func_name, "args": args, "result": result}
                                )

                            await session.send_tool_response(function_responses=function_responses)

                except Exception as e:
                    logger.exception("Receive loop error")
                    await event_queue.put({"type": "error", "error": str(e)})
                finally:
                    await event_queue.put(None)

            send_audio_task = asyncio.create_task(send_audio())
            send_video_task = asyncio.create_task(send_video())
            send_text_task = asyncio.create_task(send_text())
            receive_task = asyncio.create_task(receive_loop())

            try:
                while True:
                    event = await event_queue.get()
                    if event is None:
                        break
                    if isinstance(event, dict) and event.get("type") == "error":
                        yield event
                        break
                    yield event
            finally:
                send_audio_task.cancel()
                send_video_task.cancel()
                send_text_task.cancel()
                receive_task.cancel()
