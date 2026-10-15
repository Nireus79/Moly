#!/usr/bin/env python3
"""Replays a recorded conversation (five messages of one real chat, shortened) against a running Moly and prints the replies.
Usage: replay_conversation.py http://127.0.0.1:11498   (a throwaway profile is registered; development mode only)"""
import json, sys, time, urllib.request

base = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:11436"

def call(path, body, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(base + path, data=json.dumps(body).encode(), headers=headers)
    return json.load(urllib.request.urlopen(req, timeout=900))

token = call("/api/auth/register", {"name": "Replay", "email": f"replay{int(time.time())}@example.invalid", "password": "replay-password-123"})["token"]
conversation = ""
messages = [
    "Hello Moly. There is a matter I want to talk about. I am deeply concerned that I may loose my job soon. The company I work for more than ten years is in trouble. Sales are lower every month and I see people getting fired every week. My boss feels colder every time i try to speak to him",
    "His name is Kyle. He took over after his father retired. He was a good boss. He found the company from scratch and took care of it and us. Kyle is a college boy who never worked before. He cares only about his cars. He says that our problem is costs and we need to lower it. He doesn't care about the job. All last month, everytime I tried to talk to him he said he was busy. He totally ignored me.",
    "He changed through the time. When his father was still around he was much better.",
    "His father didn't passed away. He is not around the business any more.",
    "Alex",
]
for message in messages:
    started = time.time()
    answer = call("/api/message-processor", {"message": message, "conversationId": conversation, "browserSessionId": "replay"}, token)
    conversation = answer.get("conversationId") or conversation
    print("USER:", message[:90].replace("\n", " "))
    print("MOLY:", answer.get("response"))
    print(f"elapsed: {time.time() - started:.0f} s")
    print()
