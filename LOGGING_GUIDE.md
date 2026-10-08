# Moly Logging Guide

**Status**: ✅ PRODUCTION READY (FIX #12)  
**Date**: October 5, 2026

---

## Quick Start

### Using Structured Logging

```go
import "github.com/sirupsen/logrus"

// Logger is globally available as 'Logger'
// It's initialized in main() with config log level

// Simple info message
Logger.Info("Message processed successfully")

// Info with context (structured fields)
Logger.WithFields(logrus.Fields{
    "component": "orchestrator",
    "message_id": "msg-123",
    "layers_completed": 7,
    "duration_ms": 145.5,
}).Info("Message processing complete")

// Warning with fields
Logger.WithFields(logrus.Fields{
    "component": "cache",
    "confidence": 0.75,
    "threshold": 0.85,
}).Warn("Cache confidence below threshold")

// Error with context
Logger.WithFields(logrus.Fields{
    "component": "extraction",
    "user_id": "user-456",
    "message_id": "msg-789",
}).WithError(err).Error("Failed to extract entities")

// Debug (only shown if log level is debug)
Logger.WithFields(logrus.Fields{
    "component": "layer1",
    "entity_count": 5,
}).Debug("Extracted entities from message")
```

---

## Log Levels

### When to Use Each Level

**ERROR** - Something failed that needs attention
```go
Logger.WithError(err).WithField("component", "database").Error("Failed to connect")
```

**WARN** - Something unexpected but recovered
```go
Logger.WithField("cache_hit_ratio", 0.45).Warn("Low cache hit ratio")
```

**INFO** - Normal operation milestones (default level)
```go
Logger.WithField("layers_completed", 11).Info("Message processing complete")
```

**DEBUG** - Detailed diagnostic information (only when log_level: "debug")
```go
Logger.WithField("confidence", 0.92).Debug("Using cached summary")
```

### Setting Log Level

**In config.yaml:**
```yaml
log_level: "debug"  # "debug", "info", "warn", "error"
```

**Via environment variable:**
```bash
export MOLY_LOG_LEVEL=debug
```

**Default**: "info"

---

## Structured Fields

### Standard Fields by Component

**Orchestrator/Layer Processing:**
```go
Logger.WithFields(logrus.Fields{
    "component": "orchestrator",        // Which part of system
    "message_id": msgID,                // Message being processed
    "user_id": userID,                  // Which user
    "layer": "layer4",                  // Which layer (if applicable)
    "layers_completed": 7,              // Progress
    "duration_ms": 145.5,               // How long it took
}).Info("Processing stage complete")
```

**Extraction/LLM Operations:**
```go
Logger.WithFields(logrus.Fields{
    "component": "extraction",
    "entity_count": 5,
    "confidence": 0.92,                 // 0-1 confidence score
    "entities": "Sarah, anxiety, support",  // What was extracted
}).Info("Entities extracted")
```

**Cache Operations:**
```go
Logger.WithFields(logrus.Fields{
    "component": "cache",
    "message_id": msgID,
    "cache_hit": true,
    "confidence": 0.90,
    "threshold": 0.85,
}).Info("Cache lookup result")
```

**Database Operations:**
```go
Logger.WithFields(logrus.Fields{
    "component": "database",
    "operation": "save_summary",       // What DB operation
    "table": "message_summaries",
    "rows_affected": 1,
    "duration_ms": 5.2,
}).Info("Database operation complete")
```

**API Handlers:**
```go
Logger.WithFields(logrus.Fields{
    "component": "api",
    "method": "POST",
    "path": "/api/message-processor",
    "status_code": 200,
    "duration_ms": 250.3,
    "user_id": userID,
}).Info("API request handled")
```

---

## Migration from log.Printf()

### Old Way (Standard Log Package)
```go
log.Printf("[Layer1] ✓ Using message summary cache for %s (%d entities, confidence=%.2f)",
    lc.MessageID, len(msgSummary.ExtractedEntities), msgSummary.Confidence)
```

### New Way (Structured Logging)
```go
Logger.WithFields(logrus.Fields{
    "component": "layer1",
    "message_id": lc.MessageID,
    "entity_count": len(msgSummary.ExtractedEntities),
    "confidence": msgSummary.Confidence,
    "action": "using_cached_summary",
}).Info("Using cached message summary")
```

### Benefits
- ✅ Machine-parseable (JSON output)
- ✅ Queryable with jq
- ✅ Filterable by component/level
- ✅ Timestamps and context included automatically
- ✅ Easy to aggregate and alert on

---

## Log Output Formats

### Development (Console + File)
- Logs go to stdout and `~/.moly/moly.log`
- Readable format in console, JSON in file
- Use `tail -f ~/.moly/moly.log | jq` for structured viewing

### Production
- JSON-formatted logs to file
- Automated rotation: 100MB max per file, 3 backups, 7-day expiration
- Parse with: `cat ~/.moly/moly.log | jq '.component'` (unique components)

### Log File Rotation
- **Max size per file**: 100 MB
- **Max backups kept**: 3 files
- **Auto-delete after**: 7 days
- **Compression**: Enabled (removes old logs after archiving)

---

## Analyzing Logs

### View All Logs in Real-Time
```bash
tail -f ~/.moly/moly.log | jq
```

### Find Errors
```bash
cat ~/.moly/moly.log | jq 'select(.level=="error")'
```

### Find by Component
```bash
cat ~/.moly/moly.log | jq 'select(.component=="orchestrator")'
```

### Find by Message ID
```bash
cat ~/.moly/moly.log | jq 'select(.message_id=="msg-123")'
```

### Find by User
```bash
cat ~/.moly/moly.log | jq 'select(.user_id=="user-456")'
```

### Get Performance Metrics
```bash
cat ~/.moly/moly.log | jq 'select(.duration_ms) | {component, duration_ms}' | jq -s 'group_by(.component) | map({component: .[0].component, avg_ms: (map(.duration_ms) | add / length)})'
```

### Count Errors by Component
```bash
cat ~/.moly/moly.log | jq 'select(.level=="error") | .component' | sort | uniq -c
```

---

## Best Practices

### DO ✅
- Include `component` field (identifies which part logged)
- Include context for every log (user_id, message_id, etc)
- Use lowercase field names
- Use structured fields instead of string formatting
- Log at appropriate level (ERROR for failures, INFO for milestones)

### DON'T ❌
- Use log.Printf() - use Logger instead
- Log passwords, tokens, or sensitive data
- Use uppercase in field names (JSON convention is lowercase)
- String format instead of structured fields
- Log spam - every few seconds is OK, every millisecond is not

---

## Troubleshooting

### Logs Not Showing
**Check log level:**
```yaml
# config.yaml - should be "debug" to see DEBUG logs
log_level: "debug"
```

### Logs Not Writing to File
**Check file permissions:**
```bash
ls -la ~/.moly/
# Should show moly.log with write permissions
```

**Check disk space:**
```bash
df -h ~/.moly/
# Make sure there's space for new log files
```

### Too Much Output
**Reduce log level:**
```yaml
log_level: "warn"  # Only warnings and errors
```

### Can't Parse Logs
**Install jq:**
```bash
# macOS
brew install jq

# Linux
apt-get install jq
```

---

## Configuration

**Default config file location:**
- macOS: `~/.config/moly/config.json`
- Linux: `~/.config/moly/config.json`

**Example config.json:**
```json
{
  "port": ":11436",
  "host": "127.0.0.1",
  "log_level": "info",
  "cors_proxy_port": ":11435",
  "database_path": "~/.moly/moly.db",
  "ollama_endpoint": "http://127.0.0.1:11434"
}
```

---

## Examples by Component

### Orchestrator
```go
Logger.WithFields(logrus.Fields{
    "component": "orchestrator",
    "message_id": msg.ID,
    "user_id": msg.UserID,
    "layer": currentLayer,
    "layers_total": 11,
}).Info("Processing layer")
```

### Layer Processing
```go
Logger.WithFields(logrus.Fields{
    "component": "layer5",
    "conflict_count": len(conflicts),
    "critical_count": len(critical),
    "duration_ms": duration,
}).Info("Conflict detection complete")
```

### Database Operations
```go
Logger.WithFields(logrus.Fields{
    "component": "database",
    "table": "message_summaries",
    "operation": "insert",
    "rows_affected": 1,
    "duration_ms": 2.5,
}).Info("Save operation complete")
```

### API Requests
```go
Logger.WithFields(logrus.Fields{
    "component": "api",
    "method": "POST",
    "path": "/api/message-processor",
    "user_id": userID,
    "status_code": 200,
    "response_time_ms": 487.2,
}).Info("Request handled")
```

---

## Future Enhancements

- [ ] Structured log aggregation (all logs to central system)
- [ ] Real-time alerting (alert on ERROR logs)
- [ ] Performance dashboards (from duration_ms fields)
- [ ] User activity tracking (from user_id fields)
- [ ] Error tracking dashboard (from error logs)

---

**Last Updated**: October 5, 2026 (FIX #12 - Complete Logging System)  
**Status**: Production Ready

