# EphemeralConversationManager Implementation ✅

## Status: COMPLETE & READY

EphemeralConversationManager ties ConversationAnalyzer and ProfileUpdater together. It manages the complete lifecycle of conversations (save → extract → delete).

## What It Does

**Input**: User messages from chat  
**Process**: Save → Queue → Extract → Apply → Cleanup  
**Output**: Updated user profile

```
Conversation ends
    ↓
SaveConversation()
    ├→ conversation_ephemeral (24h TTL)
    └→ extraction_queue (status='pending')
    ↓
[Hourly] ProcessQueue()
    ├→ Fetch conversation_ephemeral
    ├→ ConversationAnalyzer.AnalyzeConversation()
    ├→ ProfileUpdater.UpdateProfile()
    └→ extraction_queue (status='completed')
    ↓
[Daily] CleanupExpired()
    └→ DELETE conversation_ephemeral (24h old)
```

## Architecture

### Main Component

**EphemeralConversationManager**
- `db`: Database connection
- `analyzer`: ConversationAnalyzer instance
- `updater`: ProfileUpdater instance

### Key Methods

**SaveConversation**
```go
func (ecm *EphemeralConversationManager) SaveConversation(
    userID string,
    conversationID string,
    messages []agents.Message,
    startedAt, endedAt int64,
) error
```
- Store conversation for 24h
- Queue for extraction
- Return immediately

**ProcessQueue** (background job)
```go
func (ecm *EphemeralConversationManager) ProcessQueue() (ProcessQueueStats, error)
```
- Fetch pending items (max 10)
- For each:
  - Analyze with ConversationAnalyzer
  - Apply with ProfileUpdater
  - Mark completed or failed
- Return stats

**CleanupExpired** (background job)
```go
func (ecm *EphemeralConversationManager) CleanupExpired() (CleanupStats, error)
```
- Delete conversations older than 24h
- Log count deleted

**Supporting Methods**
- `GetQueueStatus()` - Check queue state
- `GetEphemeralConversationCount()` - Count active conversations
- `RetryFailedExtractions()` - Retry failed queue items

## Data Flow

### 1. Save Conversation
```
User sends message through /api/v2.1/chat
    ↓
Chat handler collects conversation messages
    ↓
After conversation ends:
  SaveConversation(userID, conversationID, messages, startedAt, endedAt)
    ├→ INSERT conversation_ephemeral (expires_at = now + 24h)
    ├→ INSERT extraction_queue (status='pending')
    └→ Return (no blocking)
```

### 2. Extract & Update (Hourly Job)
```
ProcessQueue() called by scheduler
    ↓
SELECT * FROM extraction_queue WHERE status='pending' LIMIT 10
    ↓
For each item:
    ├→ UPDATE extraction_queue SET status='processing'
    ├→ Fetch messages from conversation_ephemeral
    ├→ analyzer.AnalyzeConversation(messages)
    │  └→ LLM extracts insights (AboutMe, patterns, contacts, goals)
    ├→ updater.UpdateProfile(userID, extractionResult)
    │  └→ Apply to permanent tables
    └→ UPDATE extraction_queue SET status='completed', extraction_result=JSON
```

### 3. Cleanup (Daily Job)
```
CleanupExpired() called by scheduler
    ↓
DELETE FROM conversation_ephemeral WHERE expires_at < now()
    ↓
Return count deleted
```

## Error Handling

### Queue Item Processing

If at any point extraction fails:
1. Log the error
2. Mark queue item as 'failed'
3. Increment extraction_attempt counter
4. Continue with next item

**Retry Logic**:
- Max 3 retries per item (configurable)
- Failed items can be retried via `RetryFailedExtractions()`

### Validation

**SaveConversation requires**:
- Non-empty userID
- Non-empty conversationID
- At least one message

**Returns error if validation fails**

## Performance Considerations

### Batch Processing
- ProcessQueue handles max 10 items per run
- Runs hourly via scheduler
- No blocking on chat flow

### Storage
- Conversations kept only 24 hours
- Auto-cleanup prevents disk bloat
- Compressed (JSON messages)

### Scalability
- Can handle concurrent saves (different conversations)
- Queue processing is sequential but can be parallelized later
- Extraction is I/O bound (waiting for LLM)

## Database Tables Used

### Read From
- `conversation_ephemeral` - Fetch messages

### Write To
- `conversation_ephemeral` - Insert new conversations
- `extraction_queue` - Insert queue items, update status
- (ProfileUpdater writes to permanent tables)

## Tests (8 test cases)

| Test | Verifies |
|------|----------|
| SaveConversation | Basic conversation storage |
| SaveInvalidConversation | Input validation |
| GetQueueStatus | Queue status query |
| GetConversationCount | Active conversation counting |
| CleanupExpired | Expired conversation deletion |
| RetryFailed | Failed extraction retry |
| ProcessQueue | Full extraction pipeline |
| MultipleConversations | Handling multiple conversations |

**Run tests**:
```bash
go test -v ./services -run EphemeralManagerGate
```

## Integration Points

### Receives From
- Chat handler (via SaveConversation)
- Scheduler (ProcessQueue, CleanupExpired)

### Sends To
- ConversationAnalyzer (extract)
- ProfileUpdater (apply)
- Database (store/delete)

### Dependencies
- ConversationAnalyzer (must be initialized)
- ProfileUpdater (must be initialized)
- Database with Phase 1.2 schema

## Usage Example

```go
// Initialize
analyzer := agents.NewConversationAnalyzer(llmClient, db)
updater := NewProfileUpdater(db)
manager := NewEphemeralConversationManager(db, analyzer, updater)

// Save conversation after chat ends
messages := []agents.Message{...}
err := manager.SaveConversation(
    userID,
    conversationID,
    messages,
    startedAt, endedAt)
// Returns immediately, extraction happens later

// Run in background (hourly scheduler)
stats, err := manager.ProcessQueue()
// Returns ProcessQueueStats: {Processed: 5, Failed: 1}

// Run in background (daily scheduler)
cleanup, err := manager.CleanupExpired()
// Returns CleanupStats: {DeletedCount: 42}

// Monitor
status, err := manager.GetQueueStatus()
// Returns: map[string]int{"pending": 2, "completed": 45, "failed": 1}

count, err := manager.GetEphemeralConversationCount()
// Returns: 3 active conversations
```

## Background Job Schedule

### ProcessQueue - Hourly
```go
ticker := time.NewTicker(1 * time.Hour)
defer ticker.Stop()

for range ticker.C {
    stats, err := manager.ProcessQueue()
    if err != nil {
        log.Printf("Queue processing failed: %v", err)
    } else {
        log.Printf("Processed %d, failed %d", stats.Processed, stats.Failed)
    }
}
```

### CleanupExpired - Daily
```go
ticker := time.NewTicker(24 * time.Hour)
defer ticker.Stop()

for range ticker.C {
    stats, err := manager.CleanupExpired()
    if err != nil {
        log.Printf("Cleanup failed: %v", err)
    } else {
        log.Printf("Deleted %d conversations", stats.DeletedCount)
    }
}
```

## Future Enhancements

1. **Parallel extraction**: Process queue items concurrently
2. **Priority queue**: Urgent extractions first
3. **Extraction timeout**: Configurable per-request timeout
4. **Metrics**: Extraction time, success rate, error tracking
5. **Dead letter queue**: Store permanently failed extractions for review
6. **Retry backoff**: Exponential backoff for retries

## Files

- `services/ephemeral_manager.go` - Implementation (300+ lines)
- `services/ephemeral_manager_test.go` - Tests (250+ lines)
- `database/schema_v2_1_phase_1_2.sql` - Required schema

## Next: ProfileService & HTTP Handlers

Once this is integrated, build:
1. **ProfileService** - Read API
2. **HTTP Handlers** - REST endpoints
3. **Browser Extension UI** - Display profile

