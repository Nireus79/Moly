# Μώλυ — Socratic Communication Coach

> **Better thinking before you hit send.** Moly asks the one question that matters, then helps you write. It never decides for you.

Moly is a Chrome extension with a local Go backend and a local model (Ollama, `mistral` by default). Your conversations stay on your machine.

**Status (2026-10-10):** the orchestration was rebuilt around one reply decision. Behaviour is covered by scripted end-to-end tests; live behaviour with the real model is being checked by hand (`scripts/live_chat.sh`). See [docs/ORCHESTRATOR_DESIGN.md](docs/ORCHESTRATOR_DESIGN.md) for the decisions and findings.

---

## What Moly does

- **Asks before it writes.** If something that blocks your goal is missing (who the person is, what you want), Moly asks **one** question per reply, optionally after one short sentence of what it understood.
- **A way out.** Once you have answered at least one question, a **Skip questions** button lets you take the result with what is known. Skipped questions are kept as pending and come back, ranked against new gaps, if you keep working on the same goal. A clear request with no gaps and no risk is simply answered.
- **Safety without strictness.** An unclear message is never a crisis. Moly asks first. Clear harm (self-harm, illegal acts) gets a short refusal with no help lines: *"I can't help with that. Please consider asking advice from a specialist."* (a law expert for illegal requests).
- **Doubt means ask.** A person or goal the model is not sure of is not saved; Moly confirms it with one question. A goal must be quoted from your own words.
- **Reflective questions** (Socratic) only when a model judgement says the situation has something you may not have thought through, and never before you have answered a question.
- **Your values.** Six principles in `moly-go/config/constitution.yaml` guide the questions.

## How it works (short)

Every message goes through fixed stages: safety → intent and goal → identity (the one place a person is saved) → gaps and conflicts → **one reply decision** → one transaction commit. Maturity is a single number: answered questions over answered plus open ones. Details: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Quick start

```bash
git clone https://github.com/Nireus79/Moly.git
cd Moly/moly-go
go build -o ../bin/moly .
../bin/moly            # serves on http://localhost:11436 ; Ollama must run on :11434
```

Load the extension from `moly-extension/dist` (build it with `npm install && npm run build` in `moly-extension/`). Full steps: [INSTALL.md](INSTALL.md).

## Try it

```bash
scripts/live_chat.sh      # you type every message; shows canSkip and the stage markers
scripts/live_check2.sh    # rule checks against the live model (A–F)
cd moly-go && go test -count=1 ./...   # the scripted suite
```

## Documentation

| | |
|---|---|
| [docs/ORCHESTRATOR_DESIGN.md](docs/ORCHESTRATOR_DESIGN.md) | Design, decisions, live findings (authoritative) |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Stages, data flow, where things live |
| [docs/API.md](docs/API.md) | REST API |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Build, test, live checks |
| [docs/MOLY_COMPLETE_VISION.md](docs/MOLY_COMPLETE_VISION.md) | Product vision |
| [docs/LOGGING_GUIDE.md](docs/LOGGING_GUIDE.md) | Logs (no message text or email is logged) |
| [CLAUDE.md](CLAUDE.md) | Guide for Claude Code sessions |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributing |
| [docs/archive/](docs/archive/) | Earlier plans, reports and superseded specs |

## Privacy

Stored per user in an encrypted SQLite database (key in the OS keychain): full chat history (needed to resume), contacts, questions and profile. Logs hold no message text or email. Local model by default. Known gaps: deleting a profile leaves chat messages behind; the privacy switches in `.env.example` are not read by the backend.

## License

MIT — see [LICENSE](LICENSE).
