# Troubleshooting Guide

**Version**: 1.0  
**Last Updated**: September 8, 2026

---

## Quick Diagnostic

Run these commands to check health:
```bash
# 1. Backend responding?
curl http://localhost:8080/api/v2/health

# 2. Database accessible?
sqlite3 ~/.config/moly/moly.db ".tables"

# 3. LLM available?
curl http://localhost:11434/api/tags  # If using Ollama

# 4. Check logs
tail -50 ~/.local/share/moly/moly.log
# Or if using systemd
sudo journalctl -u moly -n 50 --no-pager
```

---

## Common Issues & Solutions

### Issue: "Connection refused" / Backend Not Running

**Error Message**:
```
curl: (7) Failed to connect to localhost port 8080: Connection refused
```

**Causes**:
- Backend not started
- Wrong port
- Bound to wrong address

**Solutions**:
```bash
# Check if running
ps aux | grep moly-backend

# Start backend
cd moly-go
go run main_v2.go

# If still not working, check port
lsof -i :8080
# If something else is using it, use different port:
go run main_v2.go --port 8081
```

---

### Issue: "Database is locked"

**Error Message**:
```
[ERROR] database is locked
```

**Causes**:
- Multiple backend instances running
- Database file corrupted
- Permission issues

**Solutions**:
```bash
# Check running processes
ps aux | grep moly-backend

# Kill other instances
pkill -f moly-backend

# Wait 5 seconds
sleep 5

# Restart backend
cd moly-go
go run main_v2.go

# If still failing, reset database:
rm ~/.config/moly/moly.db
go run main_v2.go  # Will create fresh database
```

---

### Issue: "No LLM provider" / Heuristic Mode

**In Logs**:
```
[WARN] LLM initialization failed, using heuristic mode
[INFO] LLM: Heuristic Mode (Fallback)
```

**Causes**:
- Ollama not running
- API key not set
- Wrong LLM provider configured

**Solutions**:

**For Ollama**:
```bash
# Check if running
ps aux | grep ollama

# Start Ollama
ollama serve &
sleep 2

# Download model
ollama pull mistral

# Verify it works
curl http://localhost:11434/api/tags

# Restart backend with Ollama provider
export MOLY_LLM_PROVIDER=ollama
go run main_v2.go
```

**For Claude API**:
```bash
# Check if key is set
echo $ANTHROPIC_API_KEY

# If empty, set it
export ANTHROPIC_API_KEY="sk-ant-..."

# Restart backend
export MOLY_LLM_PROVIDER=claude
go run main_v2.go
```

**For OpenAI API**:
```bash
# Check key
echo $OPENAI_API_KEY

# If empty, set it
export OPENAI_API_KEY="sk-..."

# Restart backend
export MOLY_LLM_PROVIDER=openai
go run main_v2.go
```

---

### Issue: "LLM request timeout"

**Error Message**:
```
[ERROR] context deadline exceeded
[WARN] LLM returned no suggestions, using contextual fallback
```

**Causes**:
- Ollama running but slow
- Network latency
- LLM overloaded
- System too slow

**Solutions**:
```bash
# This is EXPECTED behavior on slow systems
# Backend automatically falls back to heuristics
# You'll see: "Generated 3 suggestions using heuristic fallback"

# To verify it's working:
curl -i http://localhost:8080/api/v2/conversation/generate \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "test",
    "conversationId": "test",
    "userMessage": "I need help"
  }'

# Look for:
# "phase": "context_gathering"
# "questions": [...]

# If using Ollama, make it faster:
# 1. Use smaller model: ollama pull orca-mini
# 2. Add more RAM
# 3. Use GPU acceleration
```

---

### Issue: "API returns 400 Bad Request"

**Error Message**:
```
{"error": "Missing required fields: conversationId, userId, userMessage"}
```

**Causes**:
- Missing required fields
- Wrong JSON format
- Typo in field name

**Solutions**:
```bash
# Required fields for conversation/generate:
{
  "userId": "string",           # Required
  "conversationId": "string",   # Required
  "userMessage": "string"       # Required
}

# Example correct request:
curl -X POST http://localhost:8080/api/v2/conversation/generate \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user123",
    "conversationId": "conv456",
    "userMessage": "I need help with something"
  }'
```

---

### Issue: "Database file is corrupted"

**Error Messages**:
```
database disk image malformed
database structure corrupted
```

**Solutions**:
```bash
# Option 1: Check database integrity
sqlite3 ~/.config/moly/moly.db "PRAGMA integrity_check;"

# If it says "ok", database is fine

# Option 2: Dump and restore
sqlite3 ~/.config/moly/moly.db ".dump" > backup.sql

# Option 3: If corruption confirmed, delete and rebuild
rm ~/.config/moly/moly.db

# Restart backend to recreate
cd moly-go
go run main_v2.go

# Existing user data is gone, but database is fresh
```

---

### Issue: "Too many open files"

**Error Message**:
```
too many open files
```

**Causes**:
- Resource leak
- Connection pool exhausted

**Solutions**:
```bash
# Increase file descriptor limit
ulimit -n 4096

# Or permanently in /etc/security/limits.conf
echo "moly soft nofile 4096" | sudo tee -a /etc/security/limits.conf

# Restart backend
cd moly-go
go run main_v2.go
```

---

### Issue: "Permission denied" on database file

**Error Message**:
```
permission denied: ~/.config/moly/moly.db
```

**Causes**:
- Wrong ownership
- Wrong permissions

**Solutions**:
```bash
# Fix permissions
chmod 644 ~/.config/moly/moly.db
chmod 755 ~/.config/moly

# If running as different user, fix ownership:
sudo chown moly:moly ~/.config/moly/moly.db

# Check permissions
ls -la ~/.config/moly/
```

---

### Issue: "Out of memory"

**Error Message**:
```
fatal error: runtime: out of memory
```

**Causes**:
- Memory leak in backend
- Too many concurrent connections
- Large database queries

**Solutions**:
```bash
# Monitor memory usage
watch -n 1 'ps aux | grep moly-backend'

# If memory keeps growing, restart backend:
pkill -f moly-backend
sleep 5
cd moly-go
go run main_v2.go

# To increase available memory for process:
# Edit systemd service and add:
# MemoryLimit=512M
# MemoryAccounting=yes
```

---

### Issue: "Backend crashes on startup"

**Error Message**:
```
panic: runtime error
goroutine 1 [running]
```

**Causes**:
- Database corruption
- Invalid configuration
- Missing dependencies

**Solutions**:
```bash
# 1. Check what's wrong
cd moly-go
go run main_v2.go 2>&1 | head -20

# 2. Check database
sqlite3 ~/.config/moly/moly.db ".tables"

# 3. If database issue, reset it:
rm ~/.config/moly/moly.db

# 4. Check environment variables:
echo $MOLY_LLM_PROVIDER
echo $MOLY_DB_PATH
echo $MOLY_LOG_LEVEL

# 5. Try with minimal config:
unset MOLY_LLM_PROVIDER
unset MOLY_DB_PATH
go run main_v2.go
```

---

### Issue: "Extension can't connect to backend"

**Error Message**:
```
Failed to connect to localhost:8080
```

**Causes**:
- Backend not running
- CORS headers missing
- Extension using wrong URL
- Network/firewall issue

**Solutions**:
```bash
# 1. Check backend is running
curl http://localhost:8080/api/v2/health

# 2. Check CORS headers
curl -i http://localhost:8080/api/v2/conversation/generate

# Should see headers like:
# Access-Control-Allow-Origin: *
# Access-Control-Allow-Methods: GET, POST, ...

# 3. Check extension configuration
# Extension should use: http://localhost:8080

# 4. If on different machine, use:
# http://<backend-ip>:8080
# And start backend with:
export MOLY_BIND_ADDR=0.0.0.0
go run main_v2.go

# 5. Check firewall
sudo ufw allow 8080  # If using UFW
# Or check iptables:
sudo iptables -L | grep 8080
```

---

## Performance Issues

### Slow Conversation Response (>5 seconds)

**Causes**:
- LLM is slow (expected)
- Database queries slow
- System resources low

**Diagnosis**:
```bash
# Check logs for timing
journalctl -u moly -n 100 | grep "duration"

# Monitor system
top -u moly

# Check database size
ls -lh ~/.config/moly/moly.db

# Too large (>500MB)? Run maintenance:
sqlite3 ~/.config/moly/moly.db "VACUUM;"
```

### High CPU Usage

**Causes**:
- Busy waiting loops
- Excessive logging
- Inefficient LLM usage

**Solutions**:
```bash
# Reduce log level
export MOLY_LOG_LEVEL=warn
go run main_v2.go

# Check for infinite loops
strace -p $(pgrep moly-backend) 2>&1 | head -100

# If stuck, restart:
pkill -f moly-backend
```

---

## Testing

### Run All Tests
```bash
cd moly-go
go test ./... -v -timeout 30s
```

### Test Database
```bash
go test ./... -run Database -v
```

### Test API Handlers
```bash
go test ./... -run "TestConversation\|TestHealth\|TestCORS" -v
```

### Test With Production Settings
```bash
export MOLY_LLM_PROVIDER=ollama
go test -run Production -v -timeout 180s
```

---

## Getting Help

### Debug Logs
```bash
# Set debug logging
export MOLY_LOG_LEVEL=debug
go run main_v2.go

# Save to file
go run main_v2.go > debug.log 2>&1

# Inspect logs
tail -100 debug.log
grep "\[ERROR\]" debug.log
grep "\[WARN\]" debug.log
```

### System Information
```bash
# OS
uname -a

# Go version
go version

# Database status
sqlite3 ~/.config/moly/moly.db ".status"

# Disk space
df -h ~/.config/moly

# Memory
free -h
```

### Before Reporting Issues

Collect this information:
1. Backend logs: `tail -100 ~/.local/share/moly/moly.log`
2. System info: `uname -a`, `go version`
3. Database status: `sqlite3 ~/.config/moly/moly.db ".tables"`
4. Environment: `env | grep MOLY`
5. Steps to reproduce
6. Expected vs actual behavior

---

## Quick Recovery

If everything is broken:

```bash
# 1. Stop backend
pkill -f moly-backend

# 2. Reset to clean state
rm -rf ~/.config/moly/
rm -rf ~/.local/share/moly/logs

# 3. Start fresh
cd moly-go
go run main_v2.go

# Backend will recreate database and logs
```

**This removes all user data. Use only as last resort.**

---

## Still Having Issues?

1. Check this guide for your error message
2. Review `DEPLOYMENT.md` for setup
3. Run tests: `go test ./... -v`
4. Check logs with debug level
5. Try recovery steps above

If all else fails:
- Report issue with debug logs
- Include system information
- Describe steps to reproduce
