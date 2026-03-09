#!/usr/bin/env python3
"""
Create a task via the SAI AUROSY API.
Usage: python create-task-via-api.py [base_url] [api_key] [robot_id] [scenario_id]
Example: python create-task-via-api.py http://localhost:8080/api/v1 sk-integration-abc123 r1 patrol
"""
import json
import sys
import urllib.request

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080/api/v1"
API_KEY = sys.argv[2] if len(sys.argv) > 2 else ""
ROBOT_ID = sys.argv[3] if len(sys.argv) > 3 else "r1"
SCENARIO_ID = sys.argv[4] if len(sys.argv) > 4 else "patrol"


def main():
    if not API_KEY:
        print("Usage: python create-task-via-api.py [base_url] [api_key] [robot_id] [scenario_id]")
        sys.exit(1)

    payload = {
        "robot_id": ROBOT_ID,
        "scenario_id": SCENARIO_ID,
        "payload": {},
    }
    data = json.dumps(payload).encode()

    req = urllib.request.Request(
        f"{BASE_URL}/tasks",
        data=data,
        method="POST",
        headers={
            "X-API-Key": API_KEY,
            "Content-Type": "application/json",
        },
    )

    try:
        with urllib.request.urlopen(req) as resp:
            task = json.loads(resp.read().decode())
            print("Task created:")
            print(json.dumps(task, indent=2))
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        print(f"Error {e.code}: {body}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
