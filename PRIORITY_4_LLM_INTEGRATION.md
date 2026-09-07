# Priority 4: Real LLM Integration (Next Session - 3-4 hours)

**Database + Learning Complete: ✅**

## Implement Anthropic SDK Calls

File: `moly-go/tools/llm_client.go`

Replace placeholder CallClaude() with real API:

```go
// Current:
func (lc *LLMClient) CallClaude(prompt string) (string, error) {
    return "Claude response placeholder", nil
}

// After:
func (lc *LLMClient) CallClaude(ctx context.Context, prompt string) (string, error) {
    client := anthropic.NewClient()
    resp, err := client.Messages.New(ctx, &anthropic.MessageNewParams{
        Model: anthropic.Model_Claude_3_5_Sonnet,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(prompt),
        },
    })
    if err != nil {
        return "", err
    }
    return resp.Content[0].Text, nil
}
```

## Files to Update

1. **llm_client.go** - Real Anthropic SDK calls
2. **suggestion_generator.go** - Call LLM to generate suggestions
3. **safety_checker.go** - Call LLM to detect crisis language
4. **question_generator.go** - Call LLM for Socratic questions
5. **constitution_evaluator.go** - Call LLM to check ethics

## What This Enables

After Priority 4:
- Real AI analysis of messages
- Intelligent safety detection
- Personalized Socratic questions
- Ethics evaluation
- Full V2 vision working

## Status After Priority 4

**Core Vision Complete:**
✅ Load user context from database (Priority 1-2)
✅ Record suggestion choices and learn patterns (Priority 3)
✅ Make real LLM calls for intelligent analysis (Priority 4)

**System will then:**
1. Load user's AboutMe, contacts, history
2. Generate context-aware suggestions with LLM
3. Record which suggestions user picks
4. Next time: use learned patterns + LLM = personalized

This reaches **MVP: Thinking partner with memory**

