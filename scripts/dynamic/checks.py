"""Rule checks on one played scenario. Each returns a list of failure strings. `model` is set by run.py (a call to a small model)."""
import json, re

model = None
OPTIONAL = {"clarity_question", "gap_question", "principle_question", "persistent_question", "intent_check_question", "socratic_question"}
GREEK = re.compile(r"[Ͱ-Ͽἀ-῿]")

def _json(text):
    m = re.search(r"\{.*\}|\[.*\]", text, re.S)
    try:
        return json.loads(m.group(0)) if m else None
    except Exception:
        return None

def no_errors(sc, res):
    out = []
    for i, t in enumerate(res["turns"], 1):
        if t["error"]:
            out.append(f"turn {i}: request failed: {t['error']}")
        elif not t["reply"].strip():
            out.append(f"turn {i}: empty reply")
    return out

def one_question(sc, res):
    out, greek = [], sc["expect"].get("language") == "el"
    for i, t in enumerate(res["turns"], 1):
        n = t["reply"].count("?") + t["reply"].count(";") + (t["reply"].count(";") if greek else 0)
        limit = 2 if t["exit"] == "help" else 1  # a draft may quote a question
        if n > limit:
            out.append(f"turn {i}: {n} question marks in a {t['exit']} reply")
    return out

def budget(sc, res):
    n = sum(1 for t in res["turns"] if t["exit"] in OPTIONAL)
    lim = sc["expect"].get("questions_max", 3)
    return [f"{n} optional questions, budget {lim}"] if n > lim else []

def name_questions(sc, res):
    n = sum(1 for t in res["turns"] if t["exit"] == "name_question")
    lim = sc["expect"].get("name_questions_max", 1)
    return [f"the name was asked {n} times"] if n > lim else []

def people(sc, res):
    lim = sc["expect"].get("max_people")
    got = res["state"].get("contacts", [])
    return [f"{len(got)} people stored, at most {lim} expected"] if lim is not None and len(got) > lim else []

def result(sc, res):
    by = sc["expect"].get("result_by_turn")
    if not by:
        return []
    if any(t["exit"] == "help" for t in res["turns"][:by]):
        return []
    return [f"no result (help reply) in the first {by} turns"]

def language(sc, res):
    if sc["expect"].get("language") != "el":
        return []
    return [f"turn {i}: reply not in Greek" for i, t in enumerate(res["turns"], 1) if t["reply"].strip() and not GREEK.search(t["reply"])]

def link(sc, res):
    if not sc["expect"].get("link"):
        return []
    out = []
    if not any(t["linkNotice"] for t in res["turns"][:1]) :
        # the reply itself may say it; the judge decides
        pass
    for i, t in enumerate(res["turns"][:1], 1):  # told when the link is pasted; later turns without a link are not checked
        ans = model("A user pasted a web link. Moly's reply:\n\"" + t["reply"] + "\"\n\nDoes the reply, taken as a whole, tell the user that Moly does not read web pages, "
                    "and promise nothing about opening, reading or summarising the page? Answer ONLY yes or no.")
        if not ans.lower().startswith("yes"):
            out.append(f"turn {i}: the reply does not make clear that Moly does not read the page")
    return out

def stored_facts(sc, res):
    users = "\n".join("- " + t["user"] for t in res["turns"])
    state = json.dumps(res["state"], ensure_ascii=False)[:6000]
    ans = model("These are the messages a user wrote to a coach:\n" + users + "\n\nThis is what the coach stored (people, their traits, a summary):\n" + state +
                "\n\nList stored facts about the USER or about the PEOPLE that are wrong: a fact the messages do not support, a trait attributed to the wrong person "
                "(for instance something written in text the user pasted from someone else but stored as the user's own), or a role or relation the user never gave. "
                "Do NOT list: a relationship category (romantic, professional, family, friend, other) that fits what the user said; statements in the summary about the "
                "conversation itself (what the coach asked or proposed, what is still open); wording that only paraphrases the user. "
                "Answer ONLY JSON: {\"wrong\": [\"...\"]}")
    j = _json(ans)
    if not isinstance(j, dict):
        return ["stored-facts judge unreadable"]
    return ["stored fact not supported: " + str(w) for w in j.get("wrong", [])]

def repeated(sc, res):
    qs = [(i, t["reply"]) for i, t in enumerate(res["turns"], 1) if t["exit"] in OPTIONAL | {"name_question", "confirm_question"}]
    if len(qs) < 2:
        return []
    listing = "\n".join(f"{i}. {q}" for i, q in qs)
    ans = model("These are questions a coach asked a user in one conversation, with the turn number:\n" + listing +
                "\n\nWhich pairs ask for the same information, even in other words? Answer ONLY JSON: {\"pairs\": [[3, 5]]} (empty list if none).")
    j = _json(ans)
    if not isinstance(j, dict):
        return ["repeat judge unreadable"]
    return [f"turns {p[0]} and {p[1]} ask the same thing" for p in j.get("pairs", []) if isinstance(p, list) and len(p) == 2]

def placeholders(sc, res):
    # a draft is ready to send: no [Name] style placeholders
    out = []
    for i, t in enumerate(res["turns"], 1):
        if t["exit"] == "help" and re.search(r"\[[A-Za-z][^\]\n]{0,30}\]", t["reply"]):
            out.append(f"turn {i}: a placeholder in square brackets in the reply")
    return out

def recall(sc, res):
    # a long conversation: the last message asks for a fact given early; the reply must hold it
    want = (sc["expect"].get("recall") or {}).get("word")
    if not want or not res["turns"]:
        return []
    last = res["turns"][-1]["reply"]
    return [] if want.lower() in last.lower() else [f"the last reply does not hold the fact given early ({want})"]

ALL = [recall, placeholders, no_errors, one_question, budget, name_questions, people, result, language, link, stored_facts, repeated]

def run_all(sc, res):
    out = []
    for f in ALL:
        try:
            out += f(sc, res)
        except Exception as e:
            out.append(f"{f.__name__} crashed: {e}")
    return out
