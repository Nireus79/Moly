# FIX #33: LLM Parse Validation - Comprehensive Deployment Guide

**Status**: Infrastructure Created | Critical Paths In Progress | Systematic Rollout Ready

**Scope**: 115 json.Unmarshal calls across 28 files → Apply SafeJSONParse validation

---

## QUICK START

Every LLM response parse MUST be wrapped with validation:

### BEFORE (Vulnerable):
```go
var result map[string]interface{}
if err := json.Unmarshal([]byte(llmResponse), &result); err != nil {
    log.Printf("Parse failed: %v", err)
    return nil
}
// Dangerous: result could be empty, nil, or malformed
```

### AFTER (Safe):
```go
var result map[string]interface{}
if err := tools.SafeJSONParse("FunctionName", []byte(llmResponse), &result); err != nil {
    log.Printf("Parse failed: %v", err)
    return nil, err
}
// Safe: result is validated before use
```

---

## Available Validation Functions

All in `tools/validation.go`:

### 1. SafeJSONParse(source, data, v)
Validates JSON before unmarshaling:
- Checks data is not empty
- Checks data < 1MB
- Validates JSON starts with { or [
- Returns clear error messages
- **Use this for 90% of cases**

```go
if err := tools.SafeJSONParse("Context.Parse", []byte(resp.Content), &result); err != nil {
    return err // Descriptive error with source
}
```

### 2. ValidateParsedLLMResponse(source, response, requiredFields)
Validates parsed response has required fields:
- Checks no nil values in required fields
- Checks no empty strings in required fields
- **Use for complex responses with required structure**

```go
if err := tools.ValidateParsedLLMResponse("Layer4", result, []string{"gaps", "confidence"}); err != nil {
    return err
}
```

### 3. ParseLLMResponseWithValidation(source, jsonStr, v, requiredFields)
Combines both - parse + validate in one call:
- **Use for the most critical parse points**

```go
if err := tools.ParseLLMResponseWithValidation("Layer5", jsonStr, &result, []string{"conflicts"}); err != nil {
    return err
}
```

---

## Priority Deployment Order

### TIER 1 - CRITICAL (Done/In Progress)
- [x] agents/context_extractor.go - Using SafeJSONParse
- [x] tools/conversation_summarizer.go - Using SafeJSONParse
- [ ] agents/conversation_agent.go (2 parse points)
- [ ] agents/message_processing_state.go (2 parse points)
- [ ] agents/conversation_analyzer.go (2 parse points)

**Total Tier 1**: ~7 parse points

### TIER 2 - HIGH PRIORITY (Ready)
- [ ] agents/intent_detector.go (8 parse points) - LARGEST FILE
- [ ] agents/risk_monitor.go (1 point)
- [ ] agents/message_clarity_analyzer.go (1 point)
- [ ] tools/response_parser.go (1 point)
- [ ] tools/constitutional_evaluator.go (1 point)

**Total Tier 2**: ~12 parse points

### TIER 3 - MEDIUM PRIORITY (Can Scale)
- [ ] agents/subject_shift_detector.go (1)
- [ ] agents/incoming_message_analyzer.go (1)
- [ ] agents/answer_processor.go (1)
- [ ] tools/data_loader_helpers.go (1)
- [ ] Other files with single parse points

**Total Tier 3**: ~5 parse points

### TIER 4 - COMPLETE (Optional)
- All remaining ~91 parse points can be done systematically
- Pattern is proven and reusable
- Can be batched in 2-3 additional sessions

---

## Application Template

For each file, identify all `json.Unmarshal` calls:

```bash
grep -n "json.Unmarshal" agents/intent_detector.go
```

Replace pattern:
```go
// OLD
if err := json.Unmarshal([]byte(response), &result); err != nil {

// NEW  
if err := tools.SafeJSONParse("FunctionName", []byte(response), &result); err != nil {
```

---

## Files Ready for Bulk Update

### intent_detector.go (8 parse points - PRIORITY)
Search for lines: 490, 779, 962, 1355, 1361, 1532, 1555, 1628

```go
// Line 490 example (in ParseResponse)
-  if err := json.Unmarshal([]byte(response), &parsedResp); err == nil {
+  if err := tools.SafeJSONParse("IntentDetector.ParseResponse", []byte(response), &parsedResp); err == nil {
```

### conversation_agent.go (2 parse points)
Lines: 3394, 3595

### message_processing_state.go (2 parse points)
Lines: 235, 258

### conversation_analyzer.go (2 parse points)
Lines: 169, 323

---

## Validation Checklist

Before committing each file update:

- [ ] Identified all json.Unmarshal calls in file
- [ ] Replaced with tools.SafeJSONParse or ValidateParsedLLMResponse
- [ ] Added meaningful "source" name for each parse point
- [ ] Ran `go build ./...` - clean compile
- [ ] No unused imports remaining
- [ ] Commit message references FIX #33

---

## Expected Build Result

✅ All 115+ parse points wrapped with validation  
✅ Clear error messages for malformed LLM responses  
✅ Defense-in-depth against invalid JSON  
✅ Graceful fallbacks for bad responses  
✅ Comprehensive logging for debugging  

---

## Next Steps (After This Session)

1. **Session 32 Continuation**: 
   - Apply to Tier 1 remaining (4-5 files)
   - Apply to Tier 2 (intent_detector + 4 others)
   - 2-3 commits, ~3-4 hours

2. **Session 33**:
   - Apply to Tier 3 files
   - Batch update remaining ~91 parse points
   - Comprehensive LLM validation complete

3. **Result**: 
   - 100% of LLM parse points protected
   - Moly resilient to malformed LLM responses
   - Production-ready error handling

---

## Implementation Statistics

**Infrastructure Created**:
- SafeJSONParse: Covers 90% of cases
- ValidateParsedLLMResponse: Covers structured responses
- ParseLLMResponseWithValidation: One-liner for complex cases

**Files Ready for Update**: 28  
**Parse Points to Secure**: 115  
**Status**: Foundation + infrastructure complete, ready for systematic rollout

**Effort Estimate**:
- Tier 1-2: 4-5 hours (20 parse points)
- Tier 3-4: 6-8 hours (95 parse points)
- Total: ~10-12 hours to 100% completion

---

## Do NOT

❌ Use raw json.Unmarshal without SafeJSONParse wrapper  
❌ Skip error checking on LLM response parsing  
❌ Assume LLM responses have required fields  
❌ Use parsed data without validation  
❌ Leave empty fallback paths without logging  

---

**Created**: October 6, 2026 - Session 32  
**Status**: Foundation Ready | Tier 1 Started | Systematic Rollout Planned  
**Next**: Continue with Tier 1 remaining, then Tier 2
