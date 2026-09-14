# DEVELOPMENT GUIDE

## Prerequisites

- Go 1.21+
- Node.js 18+
- SQLite 3
- Git

## Project Structure

```
Moly/
├── moly-go/              # Backend (Go)
│   ├── main.go           # Server entry point
│   ├── agents/           # Message processing agents
│   ├── database/         # SQLite schema & queries
│   ├── handlers/         # HTTP request handlers
│   ├── models/           # Data types (deprecated, use schema/)
│   ├── schema/           # Canonical types
│   └── go.mod            # Go dependencies
│
├── moly-extension/       # Frontend (TypeScript/React)
│   ├── src/
│   │   ├── sidebar/      # Chat UI
│   │   ├── popup/        # Extension popup
│   │   ├── hooks/        # React hooks
│   │   └── stores/       # Zustand state management
│   ├── manifest.json     # Chrome extension manifest
│   └── package.json      # Node dependencies
│
└── docs/                 # Documentation
```

## Backend Setup

### 1. Initialize database

```bash
cd moly-go
sqlite3 moly.db < database/schema.sql
```

### 2. Install Go dependencies

```bash
cd moly-go
go mod download
```

### 3. Build

```bash
cd moly-go
go build -o moly_server .
```

### 4. Run

```bash
cd moly-go
./moly_server
```

Server runs on `http://localhost:8080`

**Environment variables:**
- `PORT`: Server port (default 8080)
- `DB_PATH`: SQLite database path (default ./moly.db)

## Frontend Setup

### 1. Install dependencies

```bash
cd moly-extension
npm install
```

### 2. Build

```bash
cd moly-extension
npm run build
```

### 3. Load extension in Chrome

1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select the `moly-extension/dist` folder

**Note:** The extension starts with "Generate Code" mode. Use this to copy the extension ID, which is needed to test the backend.

## Testing Locally

### Test with Sample Data

1. **Start backend**
   ```bash
   cd moly-go
   ./moly_server
   ```

2. **Load extension in Chrome**
   - See "Load extension" steps above
   - Copy the extension ID (shown in chrome://extensions/)

3. **Register a new user**
   - In the extension: click "Register"
   - Email: test@example.com
   - Name: Test User
   - Password: testpass123

4. **Send a message**
   - In the extension chat: "I need to talk to my boss about the project delay"

5. **Verify response**
   - Backend should extract facts
   - Generate clarification questions
   - Frontend displays questions

### Debug Logging

**Backend:**
```bash
# Tail logs while server runs
tail -f /path/to/moly.log
```

**Frontend:**
- Open Chrome DevTools (F12)
- Console tab shows Zustand state changes
- Network tab shows API requests/responses

## Database

### Schema

Located in `moly-go/database/schema.sql`:
- `users` — authentication & profile
- `about_me` — communication preferences
- `contacts` — user relationships
- `pending_clarifications` — facts awaiting clarification
- `clarification_questions` — individual questions
- `clarification_answers` — user responses
- `context_attributes` — extracted facts
- `conversations` — chat history
- `sessions` — active sessions

### Migrations

Run migrations in order:
```bash
cd moly-go
sqlite3 moly.db < database/migrations/001_*.sql
sqlite3 moly.db < database/migrations/002_*.sql
# ... etc
```

### Inspect Database

```bash
sqlite3 moly.db
.tables
.schema users
SELECT * FROM users;
.quit
```

## Code Organization

### Backend Layers

1. **HTTP Layer** (`handlers/`)
   - Validates requests
   - Calls agents
   - Returns responses
   - Always filters by user_id

2. **Agent Layer** (`agents/`)
   - ConversationAgent: 5-phase processing
   - ClarificationEngine: question generation
   - Other agents: fact extraction, conflict detection

3. **Database Layer** (`database/`)
   - Query execution
   - Schema initialization
   - Migration application

4. **Schema Layer** (`schema/`)
   - Type definitions (canonical)
   - Validation logic
   - Response envelopes

### Frontend Components

1. **LoginScreen.tsx** — registration & login
2. **ChatInterface.tsx** — message input & display
3. **Hooks** — `useAuth`, `useAboutMe`, `useContacts`, etc.
4. **Stores** — Zustand state for user data

## Common Development Tasks

### Add a new question type

1. Add method to `ClarificationEngine` in `agents/clarification_engine.go`
2. Return `*schema.ClarificationQuestion` with full data
3. Existing database layer stores it automatically
4. Frontend displays via generic question component

### Add a new API endpoint

1. Create handler in `moly-go` (filter by user_id)
2. Register route in `main.go`
3. Document in docs/API.md
4. Add test case

### Modify the extraction logic

1. Edit `agents/conversation_agent.go` Phase 1
2. Update `schema/types.go` if new types needed
3. Test with sample messages

### Update the frontend UI

1. Edit component in `moly-extension/src/`
2. Run `npm run build`
3. Reload extension in Chrome (F5 or reload button)

## Build & Deploy

### Development

```bash
# Backend
cd moly-go && go build -o moly_server .

# Frontend
cd moly-extension && npm run build
```

### Production

```bash
# Backend with optimizations
cd moly-go && go build -ldflags="-s -w" -o moly_server .

# Frontend with minification
cd moly-extension && npm run build:prod
```

Deploy:
- Backend: Copy `moly_server` and `moly.db` to server
- Extension: Package and submit to Chrome Web Store

## Troubleshooting

### Backend won't start

```
Error: database is locked
→ Kill existing process: pkill -f moly_server
→ Check moly.db permissions: ls -la moly.db

Error: port 8080 already in use
→ Use different port: PORT=8081 ./moly_server

Error: schema.sql not found
→ Run from moly-go/ directory: pwd should show ".../Moly/Moly/moly-go"
```

### Extension can't connect to backend

```
CORS error in DevTools
→ Backend must have Authorization: Bearer token header
→ Check extension has valid session token
→ Verify localhost:8080 is running (curl http://localhost:8080/health)

Questions not displaying
→ Check "question" (lowercase) vs "Question" (uppercase) in JSON
→ Backend returns lowercase per json:"question" tags
```

### Questions not generating

```
Check logs:
- Backend: grep "GenerateSubjectClarification" logs
- Frontend: DevTools Network tab → Phase5Response body

Common causes:
- About Me profile missing (agent needs context)
- Contact not recognized (needs manual entry)
- Subject ambiguous (agent needs clarification)
```

## Next Steps

See ARCHITECTURE.md for how all pieces fit together.

See CONTRIBUTING.md for code guidelines and workflow.
