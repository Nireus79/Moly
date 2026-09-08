# Moly V2 Deployment Guide

**Version**: 1.0  
**Last Updated**: September 8, 2026  
**Status**: Production Ready

---

## Quick Start (2 minutes)

### Option A: With Ollama (Local LLM)
```bash
# 1. Start Ollama (if not running)
ollama serve &
sleep 2
ollama pull mistral

# 2. Run backend
cd moly-go
export MOLY_LLM_PROVIDER=ollama
go run main_v2.go

# Backend starts on http://localhost:8080
# Health check: curl http://localhost:8080/api/v2/health
```

### Option B: With Claude API
```bash
# 1. Set API key
export ANTHROPIC_API_KEY="sk-ant-..."

# 2. Run backend
cd moly-go
go run main_v2.go

# Backend starts on http://localhost:8080
```

### Option C: Heuristic Mode (No LLM)
```bash
# 1. Run backend
cd moly-go
go run main_v2.go

# Works with fallback suggestions (no personalization)
```

---

## Production Installation

### Prerequisites
- Go 1.21+ (for building from source)
- SQLite 3 (for database)
- 2GB disk space minimum
- 512MB RAM minimum

### Building

```bash
# Clone repository
git clone <repo> moly
cd moly/Moly/moly-go

# Build binary
go build -o moly-backend

# Binary is now ./moly-backend
```

### Configuration

**Environment Variables** (in order of precedence):
```bash
# LLM Provider Selection
export MOLY_LLM_PROVIDER="ollama"      # ollama | claude | openai | (none)

# Database
export MOLY_DB_PATH="~/.config/moly/moly.db"

# Server
export MOLY_PORT="8080"
export MOLY_BIND_ADDR="127.0.0.1"      # 127.0.0.1 (local) or 0.0.0.0 (network)

# Logging
export MOLY_LOG_LEVEL="info"            # debug | info | warn | error

# LLM Configuration
export ANTHROPIC_API_KEY="sk-..."       # For Claude API
export OPENAI_API_KEY="sk-..."          # For OpenAI API
export OLLAMA_ENDPOINT="http://localhost:11434"

# Timeouts
export MOLY_REQUEST_TIMEOUT="30s"
export MOLY_GRACEFUL_SHUTDOWN="15s"
```

**Command-line Flags** (override env vars):
```bash
./moly-backend \
  --port 8080 \
  --db-path ~/.config/moly/moly.db \
  --llm-provider ollama \
  --log-level info \
  --bind 127.0.0.1
```

---

## Running the Backend

### Development
```bash
cd moly-go
go run main_v2.go
```

### Production (Systemd)

Create `/etc/systemd/system/moly.service`:
```ini
[Unit]
Description=Moly Communication Agent Backend
After=network.target

[Service]
Type=simple
User=moly
WorkingDirectory=/opt/moly

Environment="MOLY_LLM_PROVIDER=ollama"
Environment="MOLY_LOG_LEVEL=info"
Environment="MOLY_PORT=8080"

ExecStart=/opt/moly/moly-backend

# Restart on failure
Restart=on-failure
RestartSec=5s

# Logging
StandardOutput=journal
StandardError=journal

# Security
PrivateTmp=yes
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable moly
sudo systemctl start moly
sudo systemctl status moly
```

View logs:
```bash
sudo journalctl -u moly -f
```

### Production (Docker)

**Dockerfile**:
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o moly-backend ./moly-go

FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /app/moly-backend /usr/local/bin/

EXPOSE 8080
ENV MOLY_LOG_LEVEL=info
ENV MOLY_BIND_ADDR=0.0.0.0

CMD ["moly-backend"]
```

Build and run:
```bash
# Build image
docker build -t moly-backend:latest .

# Run container (with Ollama on host)
docker run -d \
  --name moly-backend \
  --network host \
  -e MOLY_LLM_PROVIDER=ollama \
  -e MOLY_OLLAMA_ENDPOINT=http://host.docker.internal:11434 \
  -v ~/.config/moly:/root/.config/moly \
  moly-backend:latest

# Check logs
docker logs -f moly-backend

# Stop
docker stop moly-backend
```

---

## Health Checks

### API Health Check
```bash
curl http://localhost:8080/api/v2/health
```

Response:
```json
{
  "status": "ok",
  "database": "ok",
  "llm": "ok",
  "time": "2026-09-08T12:00:00Z"
}
```

### Systemd Health Check
```bash
systemctl is-active moly
```

---

## Database Management

### Database Location
```bash
~/.config/moly/moly.db    # Default
/var/lib/moly/moly.db     # Alternative (production)
```

### Backup Database
```bash
cp ~/.config/moly/moly.db ~/.config/moly/moly.db.backup
```

### Check Database Status
```bash
sqlite3 ~/.config/moly/moly.db ".tables"
sqlite3 ~/.config/moly/moly.db ".schema"
```

---

## Testing Deployment

### 1. Verify Server Started
```bash
curl http://localhost:8080/api/v2/health
# Should return: {"status":"ok", ...}
```

### 2. Test Complete Flow
```bash
# Generate conversation
curl -X POST http://localhost:8080/api/v2/conversation/generate \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "test_user",
    "conversationId": "test_conv",
    "userMessage": "I need help with something"
  }'

# Should return context gathering phase with questions
```

### 3. Run Integration Tests
```bash
cd moly-go
go test -v . -run "TestConversation" -timeout 10s
```

---

## Troubleshooting

### Port Already in Use
```bash
# Check what's using port 8080
lsof -i :8080

# Use different port
./moly-backend --port 8081
```

### Database Lock Error
```
database is locked
```
**Solution**: Close other connections or restart backend

### LLM Connection Failed
```
LLM initialization failed, using heuristic mode
```
**Solution**: 
- If using Ollama: `ollama serve` and `ollama pull mistral`
- If using API: Check `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`
- Backend works fine in heuristic mode

### Slow Performance
```bash
# Check logs
journalctl -u moly -n 100

# LLM calls timing out? Expected on slow systems
# Heuristic fallback should activate after 30s timeout
```

### Database File Keeps Growing
```bash
# Run SQLite vacuum (compacts database)
sqlite3 ~/.config/moly/moly.db "VACUUM;"
```

---

## Monitoring

### Log Levels
```bash
# View all logs
tail -f ~/.local/share/moly/moly.log

# Filter by component
journalctl -u moly | grep "\[LLMClient\]"
journalctl -u moly | grep "\[ConversationAgent\]"
```

### Performance Metrics

Backend logs request duration automatically:
```
2026-09-08 12:00:00 method=POST path=/api/v2/conversation/generate status=200 duration=45ms
```

Monitor for:
- **Slow requests** (>1s): Investigate LLM or database
- **Errors**: Check logs for [ERROR] tag
- **Timeouts**: Normal on slow systems if LLM is active

---

## Upgrades

### Backup Before Upgrade
```bash
cp ~/.config/moly/moly.db ~/.config/moly/moly.db.$(date +%Y%m%d)
```

### Stop Current Version
```bash
sudo systemctl stop moly
```

### Update Code
```bash
cd moly/Moly/moly-go
git pull
go build -o moly-backend
```

### Start New Version
```bash
sudo systemctl start moly
sudo systemctl status moly
```

### Verify
```bash
curl http://localhost:8080/api/v2/health
```

---

## Security Checklist

- [ ] Change default port if on public network
- [ ] Use firewall (allow only needed ports)
- [ ] Set `MOLY_BIND_ADDR=127.0.0.1` for local-only
- [ ] Use HTTPS reverse proxy in production (nginx/caddy)
- [ ] Keep API keys secure (use systemd env or .env file)
- [ ] Regular backups of database
- [ ] Monitor logs for errors/attacks
- [ ] Run with unprivileged user (systemd User=moly)

---

## Next Steps

1. ✅ Backend deployed
2. Install browser extension (Phase 1.2)
3. Set up CORS proxy (if needed for extension)
4. Configure firewall rules
5. Set up monitoring/logging

---

## Support

### Error Messages
See `TROUBLESHOOTING.md` for detailed error solutions

### Common Issues
- LLM timeout → Normal on slow systems, fallback works
- Database locked → Restart backend
- Port in use → Use `--port` flag to change

### Testing
```bash
# Run all tests
go test ./... -v -timeout 30s

# Run specific test
go test ./... -run TestConversationGenerate -v

# Run production tests only
go test ./... -run TestProduction -timeout 180s -v
```

---

**Deployment Complete!** Your Moly V2 backend is ready to serve requests.
