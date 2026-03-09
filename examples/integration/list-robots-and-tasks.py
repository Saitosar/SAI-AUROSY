#!/usr/bin/env python3
"""
List robots and tasks using the SAI AUROSY API.
Usage: python list-robots-and-tasks.py [base_url] [api_key]
Example: python list-robots-and-tasks.py http://localhost:8080/api/v1 sk-integration-abc123
"""
import json
import sys
import urllib.request

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080/api/v1"
API_KEY = sys.argv[2] if len(sys.argv) > 2 else ""


def request(path: str) -> dict:
    req = urllib.request.Request(
        f"{BASE_URL}{path}",
        headers={"X-API-Key": API_KEY},
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode())


def main():
    if not API_KEY:
        print("Usage: python list-robots-and-tasks.py [base_url] [api_key]")
        sys.exit(1)

    print("=== Listing robots ===")
    robots = request("/robots")
    print(json.dumps(robots, indent=2))

    print("\n=== Listing tasks ===")
    tasks = request("/tasks")
    print(json.dumps(tasks, indent=2))


if __name__ == "__main__":
    main()
