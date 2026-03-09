#!/usr/bin/env python3
"""
Webhook receiver for SAI AUROSY events with HMAC verification.
Install: pip install flask
Run: python app.py [--port 5000] [--secret YOUR_WEBHOOK_SECRET]
"""
import hashlib
import hmac
import json
import os
import sys

from flask import Flask, request

app = Flask(__name__)
SECRET = os.environ.get("WEBHOOK_SECRET", "")


def verify_signature(body: bytes, signature: str, secret: str) -> bool:
    if not secret or not signature.startswith("sha256="):
        return False
    expected = "sha256=" + hmac.new(
        secret.encode(), body, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)


@app.route("/webhooks/sai-aurosy", methods=["POST"])
def webhook():
    body = request.get_data()
    sig = request.headers.get("X-Webhook-Signature", "")
    event = request.headers.get("X-Webhook-Event", "")

    if SECRET and not verify_signature(body, sig, SECRET):
        return {"error": "Invalid signature"}, 401

    payload = json.loads(body) if body else {}
    print(f"[{event}] {json.dumps(payload, indent=2)}")

    # Process event (e.g. create Jira ticket, update CRM)
    # task_completed -> create ticket, etc.
    return "", 200


if __name__ == "__main__":
    port = 5000
    for i, arg in enumerate(sys.argv):
        if arg == "--port" and i + 1 < len(sys.argv):
            port = int(sys.argv[i + 1])
        elif arg == "--secret" and i + 1 < len(sys.argv):
            SECRET = sys.argv[i + 1]

    print(f"Webhook receiver on http://localhost:{port}/webhooks/sai-aurosy")
    print("Set WEBHOOK_SECRET env or --secret for HMAC verification")
    app.run(host="0.0.0.0", port=port)
