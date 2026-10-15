#!/usr/bin/env python3
"""Dynamic scenarios: plays each scenario against a running Moly backend and checks the product rules on the transcript.

  python3 scripts/dynamic/run.py --base http://127.0.0.1:11498 [--only D19,D20] [--runs 2] [--file scenarios.json] [--out DIR]

The backend must be started by the caller (throwaway HOME, claude-cli, see docs/DYNAMIC_SCENARIOS.md). A model plays the user when a
scenario has no fixed message left; it needs the `claude` CLI on a login. Scenarios with sensitive content live in private/ (git-ignored):
pass them with --file.
"""
import argparse, json, os, subprocess, sys, time, urllib.request, urllib.error
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import checks

SKIP_TEXT = "Go ahead with what you have."

def call(base, path, body=None, token=None, method=None):
    h = {"Content-Type": "application/json"}
    if token:
        h["Authorization"] = "Bearer " + token
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(base + path, data=data, headers=h, method=method or ("POST" if data is not None else "GET"))
    with urllib.request.urlopen(req, timeout=900) as r:
        return json.load(r)

def model(prompt, system="", timeout=300):
    cmd = ["claude", "-p", "--tools", "", "--no-session-persistence", "--model", "haiku"]
    if system:
        cmd += ["--system-prompt", system]
    p = subprocess.run(cmd, input=prompt, capture_output=True, text=True, timeout=timeout)
    return p.stdout.strip()

checks.model = model

def next_user_message(sc, turns):
    lines = []
    for t in turns:
        lines.append("User: " + t["user"])
        lines.append("Moly: " + t["reply"])
    prompt = ("You are simulating a user of Moly, a coach that helps people write messages. Persona: " + sc["persona"] + "\n"
              "Write in the language the persona's first message used. Conversation so far:\n" + "\n".join(lines) +
              "\n\nWrite ONLY the persona's next message (one to three short sentences), in character. Answer what Moly asked, "
              "and invent plausible details if you need them. If Moly has already given you a message you can use, write exactly DONE.")
    out = model(prompt, "You play a user in a test. Output only the user's next message.")
    return out.strip().strip('"')

def play(base, sc):
    token = call(base, "/api/auth/register", {"name": "Dyn", "email": f"d{int(time.time()*1000)}@example.invalid", "password": "dynamic-password-123"})["token"]
    conv, turns = "", []
    fixed = list(sc.get("fixed", []))
    msg = sc["opening"]
    for i in range(sc["max_turns"]):
        skip = isinstance(msg, dict) and msg.get("skip")
        text = SKIP_TEXT if skip else msg
        t0 = time.time()
        try:
            a = call(base, "/api/message-processor", {"message": text, "conversationId": conv, "browserSessionId": "dyn", "skipQuestions": bool(skip)}, token)
            err = None
        except Exception as e:  # recorded, checked as a failure
            a, err = {}, str(e)
        conv = a.get("conversationId") or conv
        md = a.get("metadata") or {}
        turns.append({"user": text, "reply": a.get("response") or "", "exit": md.get("replyExit") or md.get("replyPolicy"), "canSkip": md.get("canSkip"),
                      "linkNotice": bool(md.get("linkNotice")), "seconds": round(time.time() - t0, 1), "error": err})
        if err:
            break
        if fixed:
            msg = fixed.pop(0)
        else:
            nxt = next_user_message(sc, turns)
            if not nxt or nxt.upper().startswith("DONE"):
                break
            msg = nxt
    state = {"contacts": [], "memory": {}}
    try:
        for c in call(base, "/api/contacts", token=token).get("contacts", []):
            try:
                state["contacts"].append(call(base, f"/api/contacts/{c['id']}", token=token))
            except Exception:
                state["contacts"].append(c)
        state["memory"] = call(base, f"/api/conversations/{conv}/memory", token=token)
    except Exception as e:
        state["error"] = str(e)
    return {"id": sc["id"], "title": sc["title"], "turns": turns, "state": state}

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="http://127.0.0.1:11498")
    ap.add_argument("--file", default=os.path.join(os.path.dirname(os.path.abspath(__file__)), "scenarios.json"))
    ap.add_argument("--only", default="")
    ap.add_argument("--runs", type=int, default=1)
    ap.add_argument("--out", default="dynamic_out")
    ap.add_argument("--recheck", default="", help="run the checks again on the transcripts saved in this directory (no backend needed)")
    a = ap.parse_args()
    scs = json.load(open(a.file, encoding="utf-8"))
    only = set(x for x in a.only.split(",") if x)
    if a.recheck:
        byid = {sc["id"]: sc for sc in scs}
        for fn in sorted(os.listdir(a.recheck)):
            if not fn.endswith(".json"):
                continue
            res = json.load(open(os.path.join(a.recheck, fn), encoding="utf-8"))
            res["failures"] = checks.run_all(byid[res["id"]], res)
            print(fn, "OK" if not res["failures"] else "FAIL")
            for f in res["failures"]:
                print("    -", f)
        return
    os.makedirs(a.out, exist_ok=True)
    bad = 0
    for sc in scs:
        if only and sc["id"] not in only:
            continue
        for r in range(a.runs):
            res = play(a.base, sc)
            res["failures"] = checks.run_all(sc, res)
            json.dump(res, open(os.path.join(a.out, f"{sc['id']}_run{r+1}.json"), "w"), indent=1, ensure_ascii=False)
            exits = " ".join((t["exit"] or "?")[:8] for t in res["turns"])
            print(f"{sc['id']} run{r+1}: {len(res['turns'])} turns [{exits}] -> {'OK' if not res['failures'] else 'FAIL'}", flush=True)
            for f in res["failures"]:
                print("    -", f, flush=True)
            bad += bool(res["failures"])
    sys.exit(1 if bad else 0)

if __name__ == "__main__":
    main()
