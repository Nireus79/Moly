# Phase 1.2 Quick Fix Guide - Critical Issues

**Time to Production**: 4-5 hours for critical fixes  
**Scope**: 5 must-fix issues + 3 should-fix issues

---

## 🔴 CRITICAL FIX #1: Initialize JobScheduler in main.go

**Time**: 10 minutes  
**File**: moly-go/main.go (after v2db initialization)

**Add this code** (around line 207, after v2Server initialization):

```go
// Initialize ProfileUpdater for background jobs
profileUpdater := services.NewProfileUpdater(v2db)

// Initialize EphemeralConversationManager
ephemeralManager := services.NewEphemeralConversationManager(
    v2db,
    analyzer,  // ConversationAnalyzer with LLM provider
    profileUpdater,
)

// Initialize JobScheduler
scheduler := services.NewJobScheduler(v2db, ephemeralManager)

// Start background jobs
jobConfig := services.JobConfig{
    ExtractionInterval:   1 * time.Hour,
    CleanupInterval:      24 * time.Hour,
    MaxExtractionsPerRun: 10,
    EnableExtraction:     true,
    EnableCleanup:        true,
}

if err := scheduler.Start(jobConfig); err != nil {
    Logger.WithError(err).Warn("[Moly] Failed to start background job scheduler")
    // Don't fail startup, just warn
} else {
    Logger.Info("[Moly] Background job scheduler started")
}

// Add graceful shutdown
// In the existing signal handler, add:
// scheduler.Stop()
```

**Verify**: Check logs for `[JobScheduler] Starting background job scheduler...`

---

## 🔴 CRITICAL FIX #2: Implement Profile Merge Logic

**Time**: 45 minutes  
**File**: moly-go/services/profile_updater.go (lines 240-254)

**Replace mergeValue method** (line 240-246):

```go
func (pu *ProfileUpdater) mergeValue(userID, value string, confidence float64, now int64) error {
    // Get existing values
    var existingJSON []byte
    err := pu.db.QueryRow(
        "SELECT values FROM about_me_profile WHERE user_id = ?",
        userID,
    ).Scan(&existingJSON)
    
    if err != nil && err != sql.ErrNoRows {
        return err
    }
    
    // Parse existing values
    var values []string
    if existingJSON != nil {
        if err := json.Unmarshal(existingJSON, &values); err != nil {
            log.Printf("[ProfileUpdater] Failed to unmarshal values: %v", err)
            values = []string{}
        }
    }
    
    // Add new value if not already present
    found := false
    for _, v := range values {
        if v == value {
            found = true
            break
        }
    }
    if !found {
        values = append(values, value)
    }
    
    // Marshal back
    data, err := json.Marshal(values)
    if err != nil {
        return err
    }
    
    // Update database
    _, err = pu.db.Exec(
        "UPDATE about_me_profile SET values = ? WHERE user_id = ?",
        data, userID,
    )
    return err
}
```

**Replace mergePreference method** (line 248-254):

```go
func (pu *ProfileUpdater) mergePreference(userID, preference string, confidence float64, now int64) error {
    // Get existing preferences
    var existingJSON []byte
    err := pu.db.QueryRow(
        "SELECT preferences FROM about_me_profile WHERE user_id = ?",
        userID,
    ).Scan(&existingJSON)
    
    if err != nil && err != sql.ErrNoRows {
        return err
    }
    
    // Parse existing preferences
    preferences := make(map[string]interface{})
    if existingJSON != nil {
        if err := json.Unmarshal(existingJSON, &preferences); err != nil {
            log.Printf("[ProfileUpdater] Failed to unmarshal preferences: %v", err)
            preferences = make(map[string]interface{})
        }
    }
    
    // Merge preference (simple version - set key)
    // More sophisticated version would average scores, track changes, etc.
    preferences[preference] = map[string]interface{}{
        "confidence": confidence,
        "last_updated": now,
    }
    
    // Marshal back
    data, err := json.Marshal(preferences)
    if err != nil {
        return err
    }
    
    // Update database
    _, err = pu.db.Exec(
        "UPDATE about_me_profile SET preferences = ? WHERE user_id = ?",
        data, userID,
    )
    return err
}
```

**Verify**: Test that values and preferences actually get stored in database

---

## 🔴 CRITICAL FIX #3: Implement Goal Matching

**Time**: 30 minutes  
**File**: moly-go/services/profile_updater.go (line 443 area)

**Find the updateGoals method** and replace the TODO:

```go
// Before: just logging TODO
// log.Printf("[ProfileUpdater] TODO: Match goal '%s' to existing goals", update.GoalDescription)

// After: implement matching
func (pu *ProfileUpdater) matchExistingGoal(userID, goalDescription string) (int, error) {
    // Try to find similar existing goal (exact match first)
    query := `
        SELECT id FROM communication_goals
        WHERE user_id = ? AND goal LIKE ?
        ORDER BY last_updated DESC
        LIMIT 1
    `
    
    var id int
    err := pu.db.QueryRow(query, userID, "%"+goalDescription+"%").Scan(&id)
    if err == nil {
        return id, nil
    }
    
    if err != sql.ErrNoRows {
        return 0, err
    }
    
    // No match found
    return 0, nil
}

// Then in updateGoals, use it:
// existingGoalID, err := pu.matchExistingGoal(userID, update.GoalDescription)
// if existingGoalID > 0 {
//     // Update existing goal
// } else {
//     // Create new goal
// }
```

---

## 🔴 CRITICAL FIX #4: Wire ConversationAgent to ChatServer

**Time**: 30 minutes  
**Files**: moly-go/v2_1_chat_handlers.go + moly-go/v2_api_server.go

**Step 1**: Add ConversationAgent to ChatServer struct:

```go
// In v2_api_server.go ChatServer struct, add:
type ChatServer struct {
    // ... existing fields ...
    conversationAgent *agents.ConversationAgent
}

// In NewChatServer initialization, add:
server.conversationAgent = agents.NewConversationAgent(
    llmClient,  // Use the adapter
    v2db,
)
```

**Step 2**: Replace the stubbed response (line 138-144 in v2_1_chat_handlers.go):

```go
// OLD CODE (delete this):
// TODO: Phase 2 - Integrate RunChat() from ConversationAgent
response := &models.ChatResponse{
    Response:  "I'm here to help. Tell me more about what you're thinking.",
    Timestamp: time.Now().Unix(),
}

// NEW CODE (replace with this):
conversationContext := models.Context{
    UserID:     userID,
    AboutMe:    aboutMeProfile,
    ContactProfile: contactProfile,
    History:    history,
}

response, err := cs.conversationAgent.Run(conversationContext)
if err != nil {
    Logger.WithError(err).Warn("[Chat] ConversationAgent failed, using fallback")
    response = &models.ChatResponse{
        Response:  "I'm here to help. Tell me more about what you're thinking.",
        Timestamp: time.Now().Unix(),
    }
}
```

---

## 🔴 CRITICAL FIX #5: Implement Risk Detection & Intention Extraction

**Time**: 60 minutes  
**File**: moly-go/agents/conversation_agent.go (lines 199-212)

**Replace runRiskPhase** (lines 199-201):

```go
// OLD:
// TODO: Implement risk pattern detection
return nil, nil

// NEW:
func (ca *conversationAgent) runRiskPhase(ctx context.Context, message string) (string, error) {
    if message == "" {
        return "", nil
    }
    
    // Use RiskMonitor from tools
    input := &tools.ConstitutionEvaluatorInput{
        Message: message,
    }
    
    output, err := ca.constitutionEvaluator.Evaluate(ctx, input)
    if err != nil {
        // Graceful fallback
        return "", nil
    }
    
    if output == nil || output.RiskLevel == "safe" {
        return "", nil
    }
    
    // Return risk warning if detected
    return fmt.Sprintf("⚠️ Potential risk detected: %s", output.Reasoning), nil
}
```

**Replace runIntentionPhase** (lines 204-212):

```go
// OLD:
// TODO: Implement intention extraction
return "", nil

// NEW:
func (ca *conversationAgent) runIntentionPhase(ctx context.Context, message string) (string, error) {
    if message == "" {
        return "", nil
    }
    
    // Use LLM to extract intention
    prompt := fmt.Sprintf(`
        Analyze this message and identify the user's primary intention in ONE word:
        - celebrate
        - apologize
        - seek_help
        - clarify
        - inform
        - ask_permission
        
        Message: "%s"
        
        Respond with ONLY the intention word, nothing else.
    `, message)
    
    req := &tools.LLMRequest{
        SystemPrompt: "You are a communication intent analyzer. Be concise.",
        UserPrompt:   prompt,
        Temperature:  0.2,
        MaxTokens:    10,
    }
    
    resp, err := ca.llmClient.Call(ctx, req)
    if err != nil {
        return "", nil
    }
    
    return strings.ToLower(strings.TrimSpace(resp.Content)), nil
}
```

---

## 🟡 SHOULD FIX #1: Wire Job Handlers

**Time**: 15 minutes  
**File**: moly-go/main.go (HTTP routes section)

**Add these routes** (around line 227, with other V2 routes):

```go
// V2 Job Management endpoints
http.HandleFunc("/api/v2.1/jobs/status", v2Server.JobStatusHandler)
http.HandleFunc("/api/v2.1/jobs/metrics", v2Server.JobMetricsHandler)
http.HandleFunc("/api/v2.1/jobs/extraction/force", v2Server.ForceExtractionHandler)
http.HandleFunc("/api/v2.1/jobs/cleanup/force", v2Server.ForceCleanupHandler)
http.HandleFunc("/api/v2.1/jobs/metrics/reset", v2Server.ResetMetricsHandler)

Logger.Info("[Moly] Job management endpoints registered")
```

---

## 🟡 SHOULD FIX #2: Verify Database Schema

**Time**: 20 minutes  
**File**: moly-go/main.go (after v2db initialization)

**Add schema deployment check**:

```go
// After: v2db, err = database.Init(v2dbPath)

// Verify/deploy schema
deployer := database.NewSchemaDeployer(database.DeploymentConfig{
    DatabasePath: v2dbPath,
    BackupBefore: true,
    Verify:       true,
    Verbose:      false,
})

if err := deployer.Deploy(); err != nil {
    Logger.WithError(err).Warn("[Moly] Schema deployment warning (tables may already exist)")
}

// If tables don't exist, this will create them
Logger.Info("[Moly] Database schema verified/deployed")
```

---

## 🟡 SHOULD FIX #3: Persist Context to Profile

**Time**: 20 minutes  
**File**: moly-go/v2_1_chat_handlers.go (line 182)

**Replace the TODO** (line 182-183):

```go
// OLD:
// TODO: Persist learned context to AboutMe if available

// NEW:
// Extract and update profile with learned context
if response.ContextLearned {
    reflection := &models.Reflection{
        Characteristics: response.ExtractedCharacteristics,
        Interests:       response.ExtractedInterests,
        Status:          "approved",
    }
    
    if err := cs.profileService.AddReflection(userID, reflection); err != nil {
        Logger.WithError(err).Warn("[Chat] Failed to persist learned context")
    } else {
        Logger.WithField("userId", userID[:8]+"...").
            Info("[Chat] Context persisted to profile")
    }
}
```

---

## Verification Checklist

### After Fix #1 (JobScheduler)
- [ ] Logs show `[JobScheduler] Starting background job scheduler...`
- [ ] No error on startup
- [ ] Database has entries in extraction_queue table

### After Fix #2 (Profile Merge)
- [ ] Test: Values appear in about_me_profile.values column
- [ ] Test: Preferences appear in about_me_profile.preferences column
- [ ] Test: Duplicate values not added

### After Fix #3 (Goal Matching)
- [ ] Test: New goal with similar description updates existing goal
- [ ] Test: Completely new goal creates new row
- [ ] Test: Goal progress increments count properly

### After Fix #4 (ConversationAgent)
- [ ] Chat handler uses agent instead of hardcoded response
- [ ] Responses vary based on context
- [ ] Suggestions are generated

### After Fix #5 (Risk & Intention)
- [ ] Risk detection triggers on risky messages
- [ ] Intention is extracted and used for suggestions
- [ ] Fallback works if LLM fails

---

## Testing Commands

```bash
# After all fixes, test the full pipeline:

# 1. Trigger extraction job manually
curl -X POST http://localhost:8080/api/v2.1/jobs/extraction/force

# 2. Check job status
curl http://localhost:8080/api/v2.1/jobs/status | jq .

# 3. Check profile was updated
curl http://localhost:8080/api/v2.1/profile/user123 | jq .data.about_me

# 4. Send a message via chat
curl -X POST http://localhost:8080/api/v2.1/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"I want to celebrate my promotion with my boss"}'

# 5. Verify response uses agent logic, not hardcoded text
```

---

## Summary

| Fix | Status | Time | Impact |
|-----|--------|------|--------|
| JobScheduler init | Critical | 10 min | Background jobs run |
| Profile merge | Critical | 45 min | Updates persist |
| Goal matching | Critical | 30 min | Goals deduplicated |
| Wire agent | Critical | 30 min | Real chat responses |
| Risk/intention | Critical | 60 min | Safety & personalization |
| Wire handlers | High | 15 min | Can monitor jobs |
| Schema deploy | High | 20 min | Tables exist |
| Persist context | High | 20 min | Profile improves |

**Total time for critical fixes**: 175 minutes = ~3 hours  
**Total time for all fixes**: 230 minutes = ~4 hours

After these fixes, Phase 1.2 extraction pipeline will be fully operational.
