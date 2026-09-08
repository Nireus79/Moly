# Phase 1.1 Implementation Plan - Starting Now

**Target Date**: September 20, 2026 (13 days remaining)
**Current Status**: ~30% complete
**Critical Blocker**: LLM API integration

---

## THE SITUATION

You have excellent architecture docs and solid type definitions, but the actual implementation is scaffolded. Most agents and tools are stubs. You need to go from 30% to 100% in 13 days.

**Good news**: The hard part (design) is done. Now it's systematic implementation.

**Bad news**: 15-20 days of work estimated, only 13 days available.

**Solution**: Focus. Do the critical path first. Defer nice-to-haves to Phase 1.2.

---

## CRITICAL DEPENDENCY CHAIN

```
START HERE: LLM Integration
    ↓
Add Migration Runner (database)
    ↓
Complete Conversation Agent
    ↓
Complete Learning Agent
    ↓
Complete Context Manager
    ↓
Complete Risk Monitor
    ↓
Complete All Tools
    ↓
Wire All API Endpoints
    ↓
DONE: Phase 1.1
```

Each block depends on the ones above it. Don't skip.

---

## PRIORITY 1: LLM INTEGRATION (Today - Sep 7)

**Why first**: Everything else depends on LLM calls. 70% of features need this.

### What to do:

1. **Set up environment**
   ```bash
   # Create .env file in moly-go/
   ANTHROPIC_API_KEY=sk-ant-...your-key-here...
   ENVIRONMENT=development
   LOG_LEVEL=debug
   ```

2. **Implement tools/llm_client.go**
   - Load ANTHROPIC_API_KEY from .env
   - Create actual Claude API client
   - Add methods:
     ```go
     func (c *LLMClient) Think(prompt string) (string, error)
     func (c *LLMClient) Generate(prompt string, maxTokens int) (string, error)
     func (c *LLMClient) CallWithTools(tools []Tool, prompt string) (response, error)
     ```
   - Test: Make a simple API call and confirm it works
   - Effort: 4-6 hours

3. **Test it works**
   ```bash
   cd moly-go
   go run . --test-llm
   # Should output: "LLM API working: Claude Opus ready"
   ```

**BLOCKER CHECK**: If you can't get ANTHROPIC_API_KEY, everything stops here. Get that first.

---

## PRIORITY 2: DATABASE MIGRATION (Sep 7-8, parallel with LLM)

**Why**: Agents need persistence. Schema exists, just need runner.

### What to do:

1. **Add migration runner to main.go**
   ```go
   func runMigrations(db *sql.DB) error {
     migrations := []string{
       "001_create_base_schema.sql",
       "002_behavioral_profiles.sql",
       "003_indexes_constraints.sql",
     }
     for _, m := range migrations {
       content, _ := os.ReadFile(fmt.Sprintf("database/migrations/%s", m))
       if _, err := db.Exec(string(content)); err != nil {
         return err
       }
     }
     return nil
   }
   ```

2. **Call on startup**
   ```go
   func main() {
     db := initDatabase()
     if err := runMigrations(db); err != nil {
       log.Fatal("Migration failed:", err)
     }
     // ... rest of main
   }
   ```

3. **Verify**
   ```bash
   # Should create database.db with all tables
   ls -la database.db
   sqlite3 database.db ".tables"
   # Should show: about_me contacts conversations messages reflections users ...
   ```

**Effort**: 2-3 hours

---

## PRIORITY 3: CONVERSATION AGENT (Sep 8-10)

**Current**: 20% complete, does basic routing
**Target**: 100% complete, agentic loop with tool calling

### Current code (lines 38-96):
- ✅ Basic structure
- ❌ No agentic loop (just linear checks)
- ❌ Tools not used (initialized but unused)
- ❌ LLM not called
- ❌ Intention detection hardcoded

### Changes needed:

**Step 1: Replace hardcoded intention detection (lines 62-76)**
```go
// OLD: keyword matching
if contains(lowerMsg, "congratulat") { intention = "celebrate" }

// NEW: LLM-based
intention, err := ca.llmClient.DetectIntention(userMessage, conversationHistory)
if err != nil {
  intention = "general_support" // fallback
}
```
Effort: 1-2 hours

**Step 2: Add tool calling (lines 19-21 - currently unused)**
```go
// Call tools in sequence
safety, _ := ca.safetyChecker.Check(userMessage)
if safety.AlertType != "none" {
  return &models.ConversationResponse{
    Phase: "safety_alert",
    Safety: safety,
  }
}

ethics, _ := ca.constitutionEvaluator.Evaluate(userMessage)
if len(ethics.Violations) > 0 {
  questions, _ := ca.questionGenerator.GenerateEducational(ethics)
  return &models.ConversationResponse{
    Phase: "ethics_alert",
    Questions: questions,
  }
}
```
Effort: 2-3 hours

**Step 3: Integrate all generators (line 92 - currently hardcoded)**
```go
// OLD: generateContextualSuggestions() - hardcoded
suggestions := generateContextualSuggestions(aboutMe, contact, userMessage, intention)

// NEW: Use real generator with full context
suggestions, _ := ca.suggestionGenerator.Generate(models.SuggestionInput{
  AboutMe: aboutMe,
  ContactProfile: contactProfile,
  ConversationHistory: ctx.ConversationHistory,
  UserMessage: userMessage,
  Intention: intention,
  UserBehaviorProfile: userBehaviorProfile,
})
```
Effort: 1-2 hours

**Step 4: Add reflection extraction**
```go
reflection, _ := ca.contextExtractor.Extract(models.ExtractionInput{
  UserMessage: userMessage,
  Context: ctx,
  Suggestions: suggestions,
})

response.Reflection = reflection
```
Effort: 1 hour

**Step 5: Add learning integration**
```go
err = ca.learningAgent.RecordInteraction(models.InteractionData{
  UserID: userID,
  ConversationID: conversationID,
  UserMessage: userMessage,
  SuggestionsGenerated: len(suggestions.Suggestions),
  CreatedAt: time.Now().Unix(),
})
```
Effort: 30 minutes

**Total Effort for Conversation Agent**: 6-8 hours

---

## PRIORITY 4: LEARNING AGENT (Sep 10-11)

**Current**: 10% complete, all methods stubbed
**Target**: 100% complete, tracking user behavior

### What to implement:

**Method 1: RecordInteraction()**
```go
func (la *learningAgent) RecordInteraction(data InteractionData) error {
  // INSERT INTO user_interactions (user_id, conversation_id, user_message, ...)
  // VALUES (?, ?, ?, ...)
  
  return la.db.Insert("user_interactions", map[string]interface{}{
    "id": uuid.New().String(),
    "user_id": data.UserID,
    "conversation_id": data.ConversationID,
    "user_message": data.UserMessage,
    "suggestions_generated": data.SuggestionsGenerated,
    "created_at": data.CreatedAt,
  })
}
```
Effort: 1-2 hours

**Method 2: RecordSuggestionChoice()**
```go
func (la *learningAgent) RecordSuggestionChoice(data SuggestionChoiceData) error {
  // INSERT INTO suggestion_choices (...)
  // Also: UPDATE user_behavioral_profiles to track patterns
  
  // Save choice
  la.db.Insert("suggestion_choices", data)
  
  // Update profile with new choice data
  la.updateUserProfile()
  
  return nil
}
```
Effort: 1-2 hours

**Method 3: BuildBehavioralProfile()**
```go
func (la *learningAgent) BuildBehavioralProfile(userID string) (*UserBehavioralProfile, error) {
  // SELECT * FROM user_interactions WHERE user_id = ?
  // SELECT * FROM suggestion_choices WHERE user_id = ?
  // Compute patterns, averages, trends
  
  interactions := la.db.Query("SELECT * FROM user_interactions WHERE user_id = ?", userID)
  choices := la.db.Query("SELECT * FROM suggestion_choices WHERE user_id = ?", userID)
  
  profile := &UserBehavioralProfile{
    UserID: userID,
    CommunicationProfile: computePatterns(interactions),
    SuggestionChoices: analyzeChoices(choices),
    SuccessMetrics: computeMetrics(interactions),
  }
  
  return profile, nil
}
```
Effort: 2-3 hours

**Method 4: DetectPatterns()**
```go
func (la *learningAgent) DetectPatterns(userID string) (*UserPatterns, error) {
  // Analyze all suggestion_choices for user
  // Return: preferred tones, pick rates, modification rates, etc.
  
  choices := la.db.Query("SELECT tone FROM suggestion_choices WHERE user_id = ?", userID)
  
  patterns := &UserPatterns{
    PreferredTone: countTones(choices), // {"friendly": 0.65, "formal": 0.15, ...}
    SuggestionPickRate: calculatePickRate(choices),
    ModificationRate: calculateModificationRate(choices),
  }
  
  return patterns, nil
}
```
Effort: 1-2 hours

**Total Effort for Learning Agent**: 6-8 hours

---

## PRIORITY 5: CONTEXT MANAGER AGENT (Sep 11-12)

**Current**: 50% complete, some CRUD implemented
**Target**: 100% complete, full knowledge base

### What's missing:

**Method: GetContact()**
```go
func (cm *contextManager) GetContact(userID, contactID string) (*Contact, error) {
  return cm.db.QueryOne("SELECT * FROM contacts WHERE user_id = ? AND id = ?", userID, contactID)
}
```
Effort: 30 minutes

**Method: CreateContact()**
```go
func (cm *contextManager) CreateContact(userID string, contact *Contact) (*Contact, error) {
  contact.ID = uuid.New().String()
  contact.UserID = userID
  contact.CreatedAt = time.Now().Unix()
  contact.UpdatedAt = time.Now().Unix()
  
  return cm.db.Insert("contacts", contact)
}
```
Effort: 30 minutes

**Method: UpdateContact()**
```go
func (cm *contextManager) UpdateContact(userID, contactID string, updates Contact) error {
  updates.UpdatedAt = time.Now().Unix()
  return cm.db.Update("contacts", updates, "user_id = ? AND id = ?", userID, contactID)
}
```
Effort: 30 minutes

**Method: GetRelevantContext() - KEY METHOD**
```go
func (cm *contextManager) GetRelevantContext(conversationID, userID string) (*Context, error) {
  // Load all needed context
  aboutMe := cm.db.QueryOne("SELECT * FROM about_me WHERE user_id = ?", userID)
  
  conversation := cm.db.QueryOne("SELECT * FROM conversations WHERE id = ?", conversationID)
  contactID := conversation.ContactID
  contact := cm.db.QueryOne("SELECT * FROM contacts WHERE id = ?", contactID)
  
  messages := cm.db.Query("SELECT * FROM messages WHERE conversation_id = ? ORDER BY timestamp DESC LIMIT 20", conversationID)
  
  userProfile := cm.learningAgent.GetUserProfile(userID)
  
  return &Context{
    AboutMe: aboutMe,
    ContactProfile: contact,
    ConversationHistory: messages,
    UserBehaviorProfile: userProfile,
  }, nil
}
```
Effort: 1-2 hours

**Method: SaveReflection()**
```go
func (cm *contextManager) SaveReflection(conversationID string, reflection *Reflection) error {
  reflection.ID = uuid.New().String()
  reflection.Status = "pending_approval"
  reflection.CreatedAt = time.Now().Unix()
  
  return cm.db.Insert("reflections", reflection)
}
```
Effort: 1 hour

**Method: ApproveReflection()**
```go
func (cm *contextManager) ApproveReflection(conversationID string, reflection *Reflection) error {
  // Mark as approved
  reflection.Status = "approved"
  reflection.ApprovedAt = time.Now().Unix()
  cm.db.Update("reflections", reflection, "id = ?", reflection.ID)
  
  // Merge into contact
  contact := cm.GetContact(reflection.UserID, reflection.ContactID)
  contact.Characteristics = append(contact.Characteristics, reflection.Characteristics...)
  contact.Interests = append(contact.Interests, reflection.Interests...)
  contact.Notes += "\n" + reflection.UserIntention
  
  return cm.UpdateContact(reflection.UserID, reflection.ContactID, contact)
}
```
Effort: 1-2 hours

**Total Effort for Context Manager**: 4-6 hours

---

## PRIORITY 6: RISK MONITORING AGENT (Sep 12-13)

**Current**: 5% complete, all TODO
**Target**: 100% complete, risk detection

### What to implement:

**Method: AssessRisk()**
```go
func (rm *riskMonitor) AssessRisk(userID string, message string) (*RiskAssessment, error) {
  // Use LLM to detect patterns
  analysis, _ := rm.llmClient.AnalyzeForRisk(message)
  
  assessment := &RiskAssessment{
    RiskLevel: analysis.Level, // "clear", "low", "medium", "high", "immediate"
    Severity: analysis.Severity, // 0-10
    Pattern: analysis.Pattern, // "manipulation", "boundary", "scam", etc.
    EducationalQuestions: generateSocraticQuestions(analysis),
    Principles: releventPrinciples(analysis),
    Recommendation: analysis.Recommendation, // "proceed", "educate_first", "escalate"
  }
  
  return assessment, nil
}
```
Effort: 2 hours

**Method: DetectPatterns()**
```go
func (rm *riskMonitor) DetectPatterns(userID string) (*UserRiskProfile, error) {
  // Load all interactions for user
  interactions := rm.db.Query("SELECT * FROM user_interactions WHERE user_id = ?", userID)
  
  // Analyze for recurring patterns
  patterns := analyzeInteractions(interactions)
  
  profile := &UserRiskProfile{
    UserID: userID,
    RiskPatterns: patterns,
    HighestRisk: findHighestRisk(patterns),
    UpdatedAt: time.Now().Unix(),
  }
  
  return profile, nil
}
```
Effort: 2 hours

**Method: GenerateEducationalResponse()**
```go
func (rm *riskMonitor) GenerateEducationalResponse(risk RiskAssessment) ([]string, error) {
  // Create Socratic questions from violation
  questions := []string{
    "I'm noticing something in what you're planning. Can we explore this together?",
    "Why does " + risk.Pattern + " matter in this situation?",
    "What if you approached this differently? What would that look like?",
  }
  
  return questions, nil
}
```
Effort: 1-2 hours

**Method: TrackPattern()**
```go
func (rm *riskMonitor) TrackPattern(userID string, pattern *RiskPattern) error {
  return rm.db.Insert("risk_patterns", map[string]interface{}{
    "id": uuid.New().String(),
    "user_id": userID,
    "pattern_type": pattern.PatternType,
    "severity": pattern.Severity,
    "first_occurrence": pattern.FirstOccurrence,
    "trend": pattern.Trend,
  })
}
```
Effort: 1 hour

**Total Effort for Risk Monitor**: 6-8 hours

---

## PRIORITY 7: COMPLETE TOOLS (Sep 13-14)

These are lower priority because they're mostly LLM wrappers now.

**Tools to complete:**
- [ ] ConstitutionEvaluator: LLM-based principle checking (1-2 hours)
- [ ] QuestionGenerator: Socratic question generation (1-2 hours)
- [ ] ContextExtractor: Insight extraction (1-2 hours)
- [ ] SuggestionGenerator: Personalized suggestions (1-2 hours)
- [ ] SafetyChecker: Finalize patterns (1 hour)

**Total**: 6-8 hours

---

## PRIORITY 8: API HANDLERS (Sep 14-15)

Wire up the endpoints from v2_handlers.go

**Critical endpoints (must have):**
- [ ] POST /api/v2/conversation/generate ✅ scaffolded (2 hours to complete)
- [ ] POST /api/v2/conversation/feedback ✅ scaffolded (2 hours to complete)
- [ ] GET /api/v2/context/about-me (1 hour)
- [ ] POST /api/v2/context/about-me (1 hour)
- [ ] POST /api/v2/contacts (1 hour)
- [ ] GET /api/v2/contacts/:id (1 hour)

**Nice-to-have (Phase 1.2):**
- Reflections endpoints
- Risk endpoints
- Advanced context endpoints

**Total for critical**: 6 hours

---

## PRIORITY 9: TESTING (Sep 15-19)

- Agent unit tests: 4-6 hours
- Integration tests: 4-6 hours
- End-to-end tests: 2-3 hours
- Load testing: 2-3 hours

**Total**: 12-18 hours (do this in parallel with others)

---

## TIMELINE

### Week 1 (Sep 7-13)

**Day 1 (Sep 7) - Saturday:**
- [ ] LLM integration (6 hours)
- [ ] Database migration runner (2 hours)
- Total: 8 hours, should complete

**Day 2 (Sep 8) - Sunday:**
- [ ] Conversation Agent agentic loop (6 hours)
- Total: 6 hours

**Day 3 (Sep 9) - Monday:**
- [ ] Conversation Agent completion (2 hours)
- [ ] Learning Agent start (4 hours)
- Total: 6 hours

**Day 4 (Sep 10) - Tuesday:**
- [ ] Learning Agent completion (4 hours)
- [ ] Context Manager completion (2 hours)
- Total: 6 hours

**Day 5 (Sep 11) - Wednesday:**
- [ ] Context Manager finish (2 hours)
- [ ] Risk Monitor (4 hours)
- Total: 6 hours

**Day 6 (Sep 12) - Thursday:**
- [ ] Risk Monitor finish (2 hours)
- [ ] Tools completion (4 hours)
- Total: 6 hours

**Day 7 (Sep 13) - Friday:**
- [ ] Tools finish (2 hours)
- [ ] API handlers (4 hours)
- Total: 6 hours

### Week 2 (Sep 14-20)

**Day 8 (Sep 14) - Saturday:**
- [ ] API handlers finish (2 hours)
- [ ] Testing framework (4 hours)
- Total: 6 hours

**Day 9 (Sep 15) - Sunday:**
- [ ] Agent tests (6 hours)
- Total: 6 hours

**Day 10-12 (Sep 16-18):**
- [ ] Integration tests
- [ ] End-to-end tests
- [ ] Performance tuning
- [ ] Bug fixes
- Total: 18 hours

**Day 13 (Sep 19) - Thursday:**
- [ ] Final testing and fixes
- [ ] Verification against Phase 1.1 checklist
- Total: 4-6 hours

**Day 14 (Sep 20) - Friday:**
- [ ] Deadline! Ready for Phase 1.2

---

## SUCCESS CRITERIA (From Roadmap)

By Sep 20, you need:

- [x] All 4 agents running
- [x] Database schema finalized ✅ (already done)
- [x] API endpoints working
- [x] Local testing complete

- [ ] All tests passing
- [ ] Manual testing successful
- [ ] Error handling implemented
- [ ] Zero crashes for 24-hour run
- [ ] API latency < 2s p95

---

## HOW TO ACCELERATE (If needed)

If you fall behind:

1. **Defer to Phase 1.2:**
   - Risk Monitor (nice-to-have initially)
   - Full testing coverage
   - Docker setup
   - All API endpoints (just implement conversation/generate for Phase 1.1)

2. **Use LLM to help:**
   - Ask Claude to generate boilerplate code
   - Ask for database query patterns
   - Ask for test skeletons

3. **Parallelize:**
   - Get multiple developers on different agents
   - One person on tools while another on agents

4. **Simplify:**
   - No streaming initially (add in 1.2)
   - No advanced caching (add in 1.2)
   - No complex queries (optimize in 1.2)

---

## DOCUMENTATION ALREADY DONE

You don't need to write more docs. These exist:

- ✅ 01_VISION_AND_PHILOSOPHY.md (completed)
- ✅ 02_BACKEND_AGENT_ARCHITECTURE.md (completed)
- ✅ 03_AGENT_PROMPTS.md (completed)
- ✅ 05_API_SPECIFICATION.md (completed)
- ✅ 06_DATABASE_SCHEMA.md (completed)
- ✅ 12_IMPLEMENTATION_ROADMAP.md (completed)

Just follow them. Your job is to implement, not design.

---

## COMMITS TO MAKE

**Before starting implementation:**
```bash
git checkout -b phase-1.1-implementation
```

**Daily commits:**
- Sep 7: "feat: Add LLM integration with Claude API"
- Sep 8: "feat: Add database migration runner"
- Sep 9: "feat: Implement Conversation Agent agentic loop"
- Sep 10: "feat: Implement Learning Agent with persistence"
- Sep 11: "feat: Complete Context Manager Agent"
- Sep 12: "feat: Implement Risk Monitoring Agent"
- Sep 13: "feat: Complete all tools (safety, constitution, etc.)"
- Sep 14: "feat: Wire API handlers for Phase 1.1"
- Sep 15-19: "test: Add comprehensive test coverage"
- Sep 20: "chore: Phase 1.1 implementation complete"

---

## WHAT NOT TO DO

❌ Don't redesign architecture (it's done)
❌ Don't add new features (stick to spec)
❌ Don't optimize prematurely (make it work first)
❌ Don't skip LLM setup (everything depends on it)
❌ Don't ignore database (learning won't work without it)
❌ Don't commit incomplete agents (finish one completely)
❌ Don't merge without testing (test as you go)

---

## GETTING UNSTUCK

**Problem: "I don't know where to start"**
→ Start with LLM integration. Everything else depends on it.

**Problem: "This agent is too complex"**
→ Break it into smaller methods. Implement one method, test it, commit it.

**Problem: "The database queries don't work"**
→ Test each query in sqlite3 first. Then implement in Go.

**Problem: "The LLM API calls are failing"**
→ Check: ANTHROPIC_API_KEY set? Network working? Prompt valid? Rate limit?

**Problem: "Tests are failing"**
→ Mock the database for unit tests. Use real DB for integration tests.

**Problem: "Running out of time"**
→ Focus on conversation/generate endpoint only. Other endpoints can be Phase 1.2.

---

## FINAL ADVICE

1. **Do it in order** - Don't skip to the fun part
2. **Test as you go** - Don't do it all then test
3. **Commit daily** - Don't lose work
4. **Ask for help** - If stuck, get another developer
5. **Keep it simple** - Fancy can wait
6. **Stay focused** - Don't add features
7. **Remember the goal** - Get to Phase 1.1 complete

---

**Start with LLM. Today. Now. Don't read more docs. Go implement.**

The architecture is solid. The design is done. You have a clear path. Just follow it.

Good luck!
