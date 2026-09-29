# Complete Implementation Plan: Multi-Person Tracking + System Optimization

**Date**: Sept 29, 2026  
**Duration**: 4 weeks (phases)  
**Priority**: CRITICAL (20-40x performance improvement + multi-person tracking fix)  
**Status**: Ready for execution  

---

## Executive Summary

### What We're Fixing
1. **Multi-person tracking failure** (Christine vs "the girl" confusion)
2. **System 40+ minutes per message** (should be 1-2 minutes)
3. **Subject attribution loss** (no distinction between user and contact properties)
4. **Profile data unstructured** (stored as blob, not queryable)

### What We're Building
1. **Linguistic message parser** (grammar-based, deterministic)
2. **Consolidated entity extraction** (single call, chunked, cached)
3. **Subject-aware clarification processing** (structured attributes)
4. **Parallel response building** (5+ minute savings)
5. **LLM result caching** (avoid redundant calls)
6. **Contact deduplication** (merge duplicates)
7. **Early exit gates** (skip wasted processing)

### Implementation Timeline
- **Week 1**: Linguistic parser + message chunking (foundation)
- **Week 2**: Entity extraction consolidation + caching
- **Week 3**: Clarification processing + profile parsing
- **Week 4**: Parallelization + integration testing

---

## Week 1: Foundation (Linguistic Parser + Chunking)

### 1.1 Build Linguistic Message Parser

**Files to create**:
- `tools/linguistic_parser.go` (new)
  - LinguisticParser struct
  - Tokenization functions
  - POS tagging (simple)
  - Extraction rules (6 core rules)
  - Subject resolution

**Core rules to implement**:
```go
Rule 1: SubjectBeAdjective
  Pattern: Subject + "be" verb + Adjective
  Example: "I am dominant"

Rule 2: SubjectPreferenceObject
  Pattern: Subject + Preference verb + Object
  Example: "I like bondage"

Rule 3: NegatedPreference
  Pattern: Subject + Negation + Verb + Object
  Example: "I don't want casual sex"

Rule 4: NounPhraseObjects
  Pattern: Adjective* + Noun
  Example: "dominant male"

Rule 5: PrepositionalContext
  Pattern: Adjective + Preposition + Noun
  Example: "interested in experienced partners"

Rule 6: StructuredFormat
  Pattern: "Key: Value" (profile format)
  Example: "Genders: Female"
```

**Functions to implement**:
```go
type LinguisticParser struct {
    pronouns     map[string]string
    stopwords    []string
    beVerbs      []string
    preferVerbs  []string
    negations    []string
}

func NewLinguisticParser() *LinguisticParser
func (p *LinguisticParser) Parse(message string) []Extraction
func (p *LinguisticParser) Tokenize(text string) []string
func (p *LinguisticParser) SplitSentences(text string) []string
func (p *LinguisticParser) IsSubjectPronoun(word string) bool
func (p *LinguisticParser) IsBeVerb(word string) bool
func (p *LinguisticParser) IsPreferenceVerb(word string) bool
func (p *LinguisticParser) IsNegation(word string) bool
func (p *LinguisticParser) ResolveSubject(pronoun string) string
func (p *LinguisticParser) IsProfileFormat(line string) (key, value string, ok bool)
```

**Tests to write**:
- `tools/linguistic_parser_test.go`
  - Test Rule 1-6 with 5+ examples each
  - Test negation preservation
  - Test subject attribution
  - Test structured format parsing
  - Edge cases (complex sentences, capitals, punctuation)

**Wiring (Week 1 Phase 1)**:
- No wiring yet (standalone component)
- Tests verify correctness
- Ready to be wired in Week 2

---

### 1.2 Build Message Chunking

**Files to modify**:
- `main.go` (add preprocessing step)

**Files to create**:
- `tools/message_chunker.go` (new)

**Functions to implement**:
```go
type MessageChunk struct {
    Index     int
    Content   string
    ByteSize  int
    Type      string  // "user_message", "profile", "structured"
}

type MessagePreprocessor struct {
    maxChunkSize int  // 2000 bytes
}

func NewMessagePreprocessor() *MessagePreprocessor
func (mp *MessagePreprocessor) Preprocess(message string) []MessageChunk
func (mp *MessagePreprocessor) DetectMessageType(text string) string
func (mp *MessagePreprocessor) DetectProfileFormat(text string) []string
func (mp *MessagePreprocessor) ChunkMessage(text string, maxSize int) []MessageChunk
func (mp *MessagePreprocessor) MergeChunks(chunks []MessageChunk) string
```

**Logic**:
```
Preprocess(message) {
  1. Detect if message > 2000 bytes
  2. If yes, split into chunks:
     - User message (0-2000 bytes)
     - Profile section (2000+ bytes)
  3. Mark each chunk with type
  4. Return chunk array
}
```

**Wiring (Week 1 Phase 2)**:
- Add to `main.go` **BEFORE** entity extraction
- Location: Line 617 (after intent detection call)
- Call during message preprocessing
- Store chunks for later use

```go
// In main.go around line 617:
preprocessor := tools.NewMessagePreprocessor()
chunks := preprocessor.Preprocess(req.Message)
log.Printf("[MessageProcessor] Chunked message into %d parts", len(chunks))
```

**Tests to write**:
- `tools/message_chunker_test.go`
  - Test message < 2000 bytes (no chunking)
  - Test message > 2000 bytes (chunking)
  - Test profile format detection
  - Test chunk merging

---

### 1.3 Integration Test: Linguistic Parser + Chunking

**Create**: `tools/linguistic_parser_integration_test.go`

```go
func TestLinguisticParserWithChunks(t *testing.T) {
    // Test case from investigation: multi-person message with profile
    message := `No, we had no prior interaction. We have some things in common. 
                Take a look on what she has on her profile. 
                Genders: Female
                Roles: submissive
                Into: Bondage, Aftercare
                I am male dominant and I am also not interested in casual sex.`
    
    // Chunk it
    preprocessor := NewMessagePreprocessor()
    chunks := preprocessor.Preprocess(message)
    
    // Parse each chunk
    parser := NewLinguisticParser()
    allExtractions := []Extraction{}
    for _, chunk := range chunks {
        extractions := parser.Parse(chunk.Content)
        allExtractions = append(allExtractions, extractions...)
    }
    
    // Verify extractions
    AssertContainsExtraction(allExtractions, Extraction{
        Subject: "user",
        Property: "male dominant",
        Confidence: 0.95,
    })
    AssertContainsExtraction(allExtractions, Extraction{
        Subject: "she",
        Property: "submissive",
        Confidence: 0.95,
    })
    AssertContainsExtraction(allExtractions, Extraction{
        Subject: "contact",
        Property: "gender:Female",
        Confidence: 0.95,
    })
    AssertContainsExtraction(allExtractions, Extraction{
        Subject: "user",
        Property: "NOT casual sex",
        Confidence: 0.90,
    })
}
```

---

## Week 2: Entity Extraction Consolidation

### 2.1 Build Consolidated Entity Extractor

**Files to modify**:
- `agents/intent_detector.go` (refactor ExtractEntitiesWithClassification)

**Changes**:
1. Add fallback to linguistic parser when LLM fails
2. Add chunking support
3. Add result caching

```go
// In agents/intent_detector.go

func (lid *LLMIntentDetector) ExtractEntitiesSmartly(ctx context.Context, message string) ([]models.ExtractedEntity, error) {
    // Step 1: Preprocess and chunk
    preprocessor := tools.NewMessagePreprocessor()
    chunks := preprocessor.Preprocess(message)
    
    // Step 2: Try LLM extraction first
    allEntities := []models.ExtractedEntity{}
    var lmErr error
    
    for _, chunk := range chunks {
        entities, err := lid.ExtractEntitiesWithClassification(ctx, chunk.Content)
        if err != nil {
            log.Printf("[IntentDetector] LLM extraction failed: %v, falling back to linguistic parser", err)
            lmErr = err
            // Fall through to linguistic parser
            break
        }
        allEntities = append(allEntities, entities...)
    }
    
    // Step 3: If LLM failed, use linguistic parser (fallback)
    if lmErr != nil {
        log.Printf("[IntentDetector] Using linguistic parser fallback")
        parser := tools.NewLinguisticParser()
        
        for _, chunk := range chunks {
            linguisticExtractions := parser.Parse(chunk.Content)
            
            // Convert LinguisticExtraction to ExtractedEntity
            for _, lex := range linguisticExtractions {
                allEntities = append(allEntities, models.ExtractedEntity{
                    Value:      lex.Property,
                    Type:       "extracted",
                    Subject:    lex.Subject,
                    Confidence: lex.Confidence,
                    IsAmbiguous: false,
                })
            }
        }
    }
    
    return allEntities, nil
}
```

**Wiring (Week 2 Phase 1)**:
- Replace existing `ExtractEntitiesAndAnalyzeIntent` calls
- Location: `main.go` line 627

```go
// OLD (in main.go line 627):
intents, extractErr := srv.intentDetector.ExtractEntitiesAndAnalyzeIntent(context.Background(), req.Message)

// NEW:
entities, entityErr := srv.intentDetector.ExtractEntitiesSmartly(context.Background(), req.Message)
if entityErr == nil {
    log.Printf("[MessageProcessor] ✓ Extracted %d entities using smart extraction", len(entities))
} else {
    log.Printf("[MessageProcessor] ⚠ Entity extraction fully failed: %v", entityErr)
}
```

**Tests to write**:
- `agents/intent_detector_smart_test.go`
  - Test LLM path (mock LLM success)
  - Test fallback path (mock LLM timeout)
  - Test chunking
  - Test subject preservation

---

### 2.2 Implement LLM Result Caching

**Files to create**:
- `tools/llm_cache.go` (new)

```go
type LLMCache struct {
    cache map[string]string  // hash(prompt) → result
    mu    sync.RWMutex
}

func NewLLMCache() *LLMCache {
    return &LLMCache{
        cache: make(map[string]string),
    }
}

func (c *LLMCache) Get(prompt string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    result, ok := c.cache[hash(prompt)]
    return result, ok
}

func (c *LLMCache) Set(prompt, result string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache[hash(prompt)] = result
}

func (c *LLMCache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache = make(map[string]string)
}
```

**Wiring (Week 2 Phase 2)**:
- Add cache to server struct
- Location: `main.go` line 71 (ServerState struct)

```go
// In main.go ServerState:
type ServerState struct {
    // ... existing fields ...
    llmCache *tools.LLMCache  // NEW
}

// In main.go initialization (line ~140):
srv.llmCache = tools.NewLLMCache()
log.Printf("[Moly] ✓ LLM cache initialized")

// In LLM client calls, check cache first:
// (modify in tools/llm_client.go Call method)
if result, ok := c.cache.Get(req.SystemPrompt + req.UserMessage); ok {
    log.Printf("[LLMClient] Cache hit for prompt")
    return &LLMResponse{Content: result}, nil
}

// ... make LLM call ...
c.cache.Set(req.SystemPrompt + req.UserMessage, resp.Content)
```

**Tests to write**:
- `tools/llm_cache_test.go`
  - Test get/set
  - Test cache hit
  - Test cache miss
  - Test thread safety

---

### 2.3 Early Exit Gate

**Files to modify**:
- `main.go` (add decision logic)

**Changes**: Add conditional flow

```go
// In main.go around line 627 (after entity extraction):

entities, entityErr := srv.intentDetector.ExtractEntitiesSmartly(context.Background(), req.Message)

var usefulPathGate bool
if entityErr != nil {
    log.Printf("[MessageProcessor] Entity extraction failed, using simple response path")
    usefulPathGate = false
} else if len(entities) == 0 {
    log.Printf("[MessageProcessor] No entities extracted, using simple response path")
    usefulPathGate = false
} else {
    log.Printf("[MessageProcessor] Entity extraction succeeded, using full response path")
    usefulPathGate = true
}

// Later, around line 1800+ (before full analysis):
if !usefulPathGate {
    log.Printf("[MessageProcessor] Skipping deep analysis (early exit gate)")
    // Skip MessageClarityAnalyzer, topic detection, etc.
    // Go straight to simple response
    response := GenerateSimpleResponse(message)
    // Save and return
} else {
    // Continue with full analysis path
    // ... existing code ...
}
```

**Wiring (Week 2 Phase 3)**:
- Wire into main message flow
- Controls which path is taken (simple vs full)
- No new wiring needed - just conditional routing

**Tests to write**:
- `main_early_exit_test.go`
  - Test entity extraction success → full path
  - Test entity extraction failure → simple path
  - Verify simple response quality

---

## Week 3: Clarification Processing + Profile Parsing

### 3.1 Subject-Aware Clarification Processing

**Files to modify**:
- `database/clarification_capture.go` (enhance ProcessClarification)

**Changes**: Add subject parsing

```go
// In database/clarification_capture.go

func (cc *ClarificationCapture) ProcessClarificationWithSubjects(
    clarif *models.ClarificationContext,
    message string,
    userID string,
    conversationID string,
    parser *tools.LinguisticParser,  // NEW: linguistic parser
) error {
    if clarif == nil {
        return fmt.Errorf("clarification context is required")
    }

    log.Printf("[ClarificationCapture] Processing clarification type: %s", clarif.Type)

    // Step 1: Parse message to extract subject → property mappings
    extractions := parser.Parse(message)
    
    if len(extractions) == 0 {
        log.Printf("[ClarificationCapture] No extractions from linguistic parser")
        // Fallback: save entire message (old behavior)
        return cc.ProcessClarification(clarif, message, userID, conversationID)
    }

    // Step 2: Save each extraction as structured attribute (NOT blob)
    for _, extraction := range extractions {
        attr := &ContextAttribute{
            ID:             0,
            UserID:         userID,
            ConversationID: conversationID,
            FactType:       extraction.Property,
            FactValue:      extraction.Property,
            AttributedTo:   extraction.Subject,  // NEW: subject attribution
            Context:        "general",
            Confidence:     extraction.Confidence,
            Source:         "clarification_response",
            Evidence:       message,
            Version:        1,
            CreatedAt:      time.Now().Unix(),
        }

        err := cc.contextAttrRepo.Save(attr)
        if err != nil {
            log.Printf("[ClarificationCapture] Error saving attribute: %v", err)
            return err
        }
        
        log.Printf("[ClarificationCapture] ✓ Saved %s = %s (subject: %s)",
            extraction.Property, extraction.Property, extraction.Subject)
    }

    return nil
}
```

**Wiring (Week 3 Phase 1)**:
- Location: `main.go` line 703 (in clarification processing block)

```go
// OLD (in main.go line 708):
if err := clarificationCapture.ProcessClarification(clarificationType, req.Message, userID, req.ConversationID); err != nil {
    log.Printf("[MessageProcessor] Layer 3: ⚠️  Error processing clarification: %v", err)
}

// NEW:
parser := tools.NewLinguisticParser()  // Create parser instance
if err := clarificationCapture.ProcessClarificationWithSubjects(
    clarificationType, 
    req.Message, 
    userID, 
    req.ConversationID,
    parser,
) {
    log.Printf("[MessageProcessor] Layer 3: ⚠️  Error processing clarification: %v", err)
}
```

**Tests to write**:
- `database/clarification_capture_subjects_test.go`
  - Test subject parsing
  - Test structured attribute storage
  - Test multi-subject message
  - Verify queryability

---

### 3.2 Profile Parser

**Files to create**:
- `tools/profile_parser.go` (new)

```go
type ProfileParser struct {
    // Maps for Fetlife format
    profileKeys map[string]string  // "genders" → "gender"
}

func NewProfileParser() *ProfileParser {
    return &ProfileParser{
        profileKeys: map[string]string{
            "genders":     "gender",
            "gender":      "gender",
            "roles":       "role",
            "role":        "role",
            "orientation": "orientation",
            "pronouns":    "pronouns",
            "pronoun":     "pronouns",
            "into":        "interests",
            "interests":   "interests",
            "interest":    "interests",
        },
    }
}

func (p *ProfileParser) ParseFetlifeProfile(profileText string) map[string][]string {
    result := make(map[string][]string)
    
    lines := strings.Split(profileText, "\n")
    for _, line := range lines {
        if strings.Contains(line, ":") {
            parts := strings.Split(line, ":")
            if len(parts) == 2 {
                key := strings.TrimSpace(strings.ToLower(parts[0]))
                value := strings.TrimSpace(parts[1])
                
                if normalizedKey, ok := p.profileKeys[key]; ok {
                    // Split by comma for interests/into fields
                    if normalizedKey == "interests" {
                        values := strings.Split(value, ",")
                        for _, v := range values {
                            result[normalizedKey] = append(result[normalizedKey], strings.TrimSpace(v))
                        }
                    } else {
                        result[normalizedKey] = append(result[normalizedKey], value)
                    }
                }
            }
        }
    }
    
    return result
}
```

**Wiring (Week 3 Phase 2)**:
- Location: In clarification processing, after linguistic parsing
- Call to extract profile and save as contact attributes

```go
// In ProcessClarificationWithSubjects, after linguistic parsing:

// Step 3: Check if message contains profile format
profileParser := tools.NewProfileParser()
profileData := profileParser.ParseFetlifeProfile(message)

if len(profileData) > 0 {
    log.Printf("[ClarificationCapture] Detected profile format, saving as contact attributes")
    
    // Determine which contact this profile belongs to
    contactSubject := DetermineContactSubject(message)  // Extract contact name
    
    for key, values := range profileData {
        for _, value := range values {
            attr := &ContextAttribute{
                ID:             0,
                UserID:         userID,
                ConversationID: conversationID,
                FactType:       key,
                FactValue:      value,
                AttributedTo:   contactSubject,  // Who does this belong to?
                Context:        "profile",
                Confidence:     0.95,
                Source:         "profile_extraction",
                Evidence:       message,
                Version:        1,
                CreatedAt:      time.Now().Unix(),
            }
            
            cc.contextAttrRepo.Save(attr)
            log.Printf("[ClarificationCapture] ✓ Profile: %s = %s (contact: %s)",
                key, value, contactSubject)
        }
    }
}
```

**Tests to write**:
- `tools/profile_parser_test.go`
  - Test Fetlife format parsing
  - Test comma-separated interests
  - Test missing/malformed format
  - Test various key spellings

---

### 3.3 Contact Deduplication Trigger

**Files to modify**:
- `main.go` (add deduplication call after entity extraction)

**Logic**:
```go
// After entity extraction (Week 2), add deduplication

// In main.go around line 650:

if len(entities) > 0 {
    log.Printf("[MessageProcessor] Checking for contact duplicates...")
    
    deduplicator := database.NewContactDeduplicator(srv.database)
    
    for _, entity := range entities {
        if entity.Type == "contact" {
            // Check if this contact already exists
            merged, err := deduplicator.CheckAndMergeDuplicates(userID, entity.Value)
            if err != nil {
                log.Printf("[MessageProcessor] Deduplication check failed: %v", err)
                continue
            }
            
            if merged {
                log.Printf("[MessageProcessor] ✓ Contact merged: %s", entity.Value)
            }
        }
    }
}
```

**Wiring (Week 3 Phase 3)**:
- Location: `main.go` line ~650 (after entity extraction)
- Triggers ContactDeduplicator to merge Christine (msg1) + "the girl" (msg2)

**Tests to write**:
- Integration test showing merge:
  - Message 1: Extract "Christine"
  - Message 2: Extract "the girl"
  - Verify they're merged to single contact
  - Verify all characteristics consolidated

---

## Week 4: Parallelization + Integration Testing

### 4.1 Parallelize Response Building

**Files to modify**:
- `main.go` (refactor response building section)

**Changes**: Use goroutines for independent operations

```go
// In main.go around line 2450+ (response building):

// OLD (sequential):
response := generateResponse()
insights := extractInsights()
eval := evaluateResponse()
risk := assessRisk()

// NEW (parallel):
type ResponseBuilding struct {
    response models.Response
    insights models.InsightResult
    eval     models.EvaluationResult
    risk     models.RiskResult
}

building := ResponseBuilding{}
errors := make(chan error, 4)
done := make(chan bool)

// Launch 4 concurrent goroutines
go func() {
    r, err := generateResponse()
    if err != nil {
        errors <- err
        return
    }
    building.response = r
}()

go func() {
    i, err := extractInsights()
    if err != nil {
        errors <- err
        return
    }
    building.insights = i
}()

go func() {
    e, err := evaluateResponse()
    if err != nil {
        errors <- err
        return
    }
    building.eval = e
}()

go func() {
    r, err := assessRisk()
    if err != nil {
        errors <- err
        return
    }
    building.risk = r
}()

// Wait for all to complete
go func() {
    for i := 0; i < 4; i++ {
        select {
        case <-errors:
            // Continue even if one fails
        }
    }
    done <- true
}()

<-done

// Use building.response, building.insights, building.eval, building.risk
```

**Wiring (Week 4 Phase 1)**:
- Location: `main.go` line ~2450
- Replace sequential calls with parallel goroutines
- Captures all results before continuing

**Tests to write**:
- Benchmark test showing 5+ minute savings
- Test error handling (one goroutine fails)
- Test all results collected correctly

---

### 4.2 Full Integration Test

**Files to create**:
- `main_integration_test.go` (comprehensive test)

```go
func TestMultiPersonTrackingFlow(t *testing.T) {
    // Simulate complete flow from investigation example
    
    server := setupTestServer()
    defer server.Cleanup()
    
    // Message 1: User introduces Christine
    msg1 := "I want to talk about Christine from Fetlife. She's 39, female submissive."
    resp1 := server.ProcessMessage(msg1)
    
    // Verify: Christine extracted
    AssertEntityExists(resp1, "Christine", "contact")
    
    // Message 2: User provides Christine's full profile + user's info
    msg2 := `No, we had no prior interaction. We have some things in common. 
             Genders: Female. Roles: submissive. Into: Bondage, Aftercare.
             I am male dominant and not interested in casual sex.`
    
    // This should NOT timeout
    start := time.Now()
    resp2 := server.ProcessMessage(msg2)
    elapsed := time.Since(start)
    
    // Verify response time (should be 1-2 minutes, not 40+)
    if elapsed > 3*time.Minute {
        t.Errorf("Message processing took %v, expected < 3 minutes", elapsed)
    }
    
    // Verify: Correct subject attribution
    AssertExtraction(resp2, "user", "male dominant")
    AssertExtraction(resp2, "Christine", "female")
    AssertExtraction(resp2, "Christine", "submissive")
    AssertExtraction(resp2, "Christine", "interest:Bondage")
    AssertExtraction(resp2, "Christine", "interest:Aftercare")
    AssertExtraction(resp2, "user", "NOT casual sex")
    
    // Verify: No duplicate contacts
    contacts := server.GetContacts("Christine")
    if len(contacts) != 1 {
        t.Errorf("Expected 1 contact for Christine, got %d", len(contacts))
    }
    
    // Verify: Profile data queryable
    profile := server.GetContactProfile("Christine")
    AssertProfileHas(profile, "gender", "Female")
    AssertProfileHas(profile, "role", "submissive")
    AssertProfileHas(profile, "interests", []string{"Bondage", "Aftercare"})
    
    // Verify: Moly response is correct
    responseText := resp2.Message
    if !strings.Contains(responseText, "Christine") {
        t.Error("Response should mention Christine")
    }
    if strings.Contains(responseText, "you're looking for a dominant") {
        t.Error("Response should NOT confuse Christine's needs with user's")
    }
}
```

**Wiring (Week 4 Phase 2)**:
- Full end-to-end test verifying all components
- Tests the exact scenario from the investigation
- Validates fixes for:
  - Multi-person tracking
  - Performance (1-2 min vs 40+ min)
  - Subject attribution
  - Profile structuring
  - Contact deduplication

---

### 4.3 Performance Benchmarking

**Files to create**:
- `benchmark_test.go` (performance validation)

```go
func BenchmarkMessageProcessing(b *testing.B) {
    server := setupTestServer()
    defer server.Cleanup()
    
    message := `I'm dominant male 47 years old. Se is 32 female submissive. 
                Genders: Female. Roles: submissive. Into: Bondage, Aftercare.
                I don't want casual sex.`
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        start := time.Now()
        server.ProcessMessage(message)
        elapsed := time.Since(start)
        
        // Report timing
        b.Logf("Message processed in %v", elapsed)
    }
    
    // Expected: 1-2 minutes
    // Old: 40+ minutes
}
```

**Wiring (Week 4 Phase 3)**:
- Benchmark entire pipeline
- Verify 20-40x speedup
- Check all components wired

---

## Complete Wiring Checklist

### Week 1 Wiring
- [ ] Linguistic parser standalone (no wiring yet)
- [ ] Message chunker wired into preprocessing

### Week 2 Wiring
- [ ] SmartExtraction wired at line 627 in main.go
- [ ] LLM cache wired to ServerState
- [ ] Early exit gate wired (conditional routing)
- [ ] Fallback to linguistic parser on LLM failure

### Week 3 Wiring
- [ ] Subject-aware clarification at line 703 in main.go
- [ ] Profile parser integrated into clarification processing
- [ ] Contact deduplicator triggered after entity extraction
- [ ] All attributes saved with subject attribution

### Week 4 Wiring
- [ ] Parallelization at line 2450+ in main.go
- [ ] All goroutines integrated into response building
- [ ] Integration tests verify all wiring
- [ ] Benchmarks confirm performance gains

---

## File-by-File Wiring Map

```
tools/
  ├── linguistic_parser.go (NEW) ← Week 1, no external wiring
  ├── message_chunker.go (NEW) ← Week 1, wired at main.go:617
  ├── llm_cache.go (NEW) ← Week 2, wired at main.go:71
  └── profile_parser.go (NEW) ← Week 3, wired in clarification

database/
  ├── clarification_capture.go (MODIFY) ← Week 3, add ProcessClarificationWithSubjects

agents/
  └── intent_detector.go (MODIFY) ← Week 2, add ExtractEntitiesSmartly

main.go (CRITICAL WIRING)
  ├── Line 71: Add llmCache to ServerState
  ├── Line 140: Initialize llmCache
  ├── Line 617: Preprocess/chunk message
  ├── Line 627: Call ExtractEntitiesSmartly (REPLACE)
  ├── Line 650: Trigger deduplicator
  ├── Line 703: Call ProcessClarificationWithSubjects (REPLACE)
  ├── Line 1800: Apply early exit gate
  └── Line 2450: Parallelize response building
```

---

## Testing Strategy

### Unit Tests (Week-by-week)
- Week 1: Linguistic parser, chunker
- Week 2: Smart extraction, caching, early exit
- Week 3: Subject clarification, profile parser, deduplication
- Week 4: Parallelization benchmarks

### Integration Tests
- End-to-end flow (exact scenario from investigation)
- Multi-person tracking correctness
- Performance gains (1-2 min target)
- Subject attribution accuracy

### Regression Tests
- Existing clarification flow still works
- Single-person messages unaffected
- Contact extraction accuracy unchanged
- Profile data queryability

---

## Success Criteria

✅ **Performance**
- Single message processing: 40+ min → 1-2 min (20-40x)
- Parallel building: 7m40s → 2m40s (5m+ savings)
- LLM caching: Redundant calls eliminated
- Linguistic fallback: <100ms when LLM fails

✅ **Multi-Person Tracking**
- User vs contact distinction (100% accurate)
- Subject attribution on all extractions
- Negation preservation ("NOT casual sex")
- Christine + "the girl" merged to single contact

✅ **Data Quality**
- Profile data structured (queryable)
- Contact attributes with subjects
- No blob storage (all structured)
- Zero duplicate contacts

✅ **System Stability**
- No LLM timeouts (fallback works)
- All components wired
- Integration tests pass 100%
- Benchmarks confirm gains

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Linguistic parser incomplete | Weeks 1-2 for rules refinement |
| Chunking breaks entities | Week 1 integration tests |
| Wiring mistakes | Detailed wiring map + checklist |
| Performance not achieved | Benchmarks each week |
| Regression in existing flow | Regression test suite |

---

## Go-Live Checklist

- [ ] All 4 weeks of work complete
- [ ] 100% unit tests passing
- [ ] 100% integration tests passing
- [ ] Performance benchmarks verified (1-2 min target)
- [ ] Regression tests all pass
- [ ] Code review completed
- [ ] Wiring map verified (every line checked)
- [ ] Documentation updated
- [ ] Commit messages clear and complete
- [ ] Ready for production deployment

---

**Status**: Ready for Week 1 execution  
**Estimated Total Duration**: 4 weeks  
**Team**: 1-2 engineers  
**Priority**: CRITICAL (20-40x performance improvement)  

---

**Document Status**: Complete ✅  
**Next Step**: Start Week 1 (Linguistic Parser + Chunking)
