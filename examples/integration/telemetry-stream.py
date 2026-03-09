#!/usr/bin/env python3
"""
Stream telemetry from SAI AUROSY via SSE.
Usage: python telemetry-stream.py [base_url] [api_key]
Example: python telemetry-stream.py http://localhost:8080/api/v1 sk-integration-abc123
"""
import json
import sys
import urllib.request

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080/api/v1"
API_KEY = sys.argv[2] if len(sys.argv) > 2 else ""


def main():
    if not API_KEY:
        print("Usage: python telemetry-stream.py [base_url] [api_key]")
        sys.exit(1)

    req = urllib.request.Request(
        f"{BASE_URL}/telemetry/stream",
        headers={
            "X-API-Key": API_KEY,
            "Accept": "text/event-stream",
        },
    )

    with urllib.request.urlopen(req) as resp:
        buffer = b""
        for chunk in iter(lambda: resp.read(4096), b""):
            if not chunk:
                break
            buffer += chunk
            while b"\n\n" in buffer or b"\r\n\r\n" in buffer:
                sep = b"\n\n" if b"\n\n" in buffer else b"\r\n\r\n"
                event, buffer = buffer.split(sep, 1)
                lines = event.decode().strip().split("\n")
                data = None
                for line in lines:
                    if line.startswith("data: "):
                        try:
                            data = json.loads(line[6:])
                            print(json.dumps(data))
                        except json.JSONDecodeError:
                            pass
                        break


if __name__ == "__main__":
    main()
