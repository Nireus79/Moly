# LLM Call Site Inventory

## ANALYSIS: Which LLM calls need adaptation?

### ✅ NEEDS ADAPTATION (tone-dependent)
1. **generateConversationalResponse()** [conversation_agent.go]
   - Purpose: Generate main response to user
   - Current: NOW ADAPTIVE (style + emotion + topic)
   - Needs: Yes - user preferences matter for how response sounds
   - Status: ✅ IMPLEMENTED with new architecture

2. **ClarificationAsker.Ask()** [tools/clarification_asker.go]
   - Purpose: Ask for missing context
   - Current: "sound like a friend" (static)
   - Needs: Yes - formal user should get formal question
   - Status: ⏳ Could use adaptation

3. **SuggestionGenerator.Generate()** [tools/suggestion_generator.go]
   - Purpose: Generate message suggestions
   - Current: ALREADY RECEIVES (UserCommunicationStyle, Tone, Mode) - partial adaptation
   - Needs: Yes - already doing it!
   - Status: ✅ Already receives context in input

### ❌ DOES NOT NEED ADAPTATION (rules-based)
1. **ContextExtractor.Extract()** [agents/context_extractor.go]
   - Purpose: Extract facts from message (who, style, intention)
   - Current: Static prompt asking "what did user say?"
   - Needs: No - "what they said" is the same for everyone
   - Status: ✅ OK as-is

2. **HarmAnalyzer.Analyze()** [tools/harm_analyzer.go]
   - Purpose: Detect harmful/dangerous content
   - Current: Static safety rules
   - Needs: No - danger is universal, not user-preference dependent
   - Status: ✅ OK as-is

3. **SafetyChecker.Check()** [tools/safety_checker.go]
   - Purpose: Crisis/safety detection
   - Current: Static rules + keyword detection
   - Needs: No - crisis is crisis regardless of user style
   - Status: ✅ OK as-is

4. **QuestionGenerator.Generate()** [tools/question_generator.go]
   - Purpose: Generate Socratic/clarifying questions
   - Current: Receives UserCommunicationStyle in input, uses in UserPrompt (not SystemPrompt)
   - Needs: Partial - could use better adaptation
   - Status: ⚠️  Already gets style but could be more integrated

5. **ConstitionEvaluator.Evaluate()** [tools/constitution_evaluator.go]
   - Purpose: Check if content violates principles
   - Current: Static ethical rules
   - Needs: No - principles are universal
   - Status: ✅ OK as-is

### 🔍 CURRENT STATE OF CONTEXT PASSING

| Tool | Receives Style? | Uses in System? | Uses in User? | Status |
|------|-----------------|-----------------|---------------|--------|
| generateConversationalResponse | ✅ YES (extracted) | ✅ YES (ADAPTIVE) | ✅ YES | ✅ GOOD |
| SuggestionGenerator | ✅ YES (input param) | ✅ YES (line 118) | ✅ YES | ✅ GOOD |
| QuestionGenerator | ✅ YES (input param) | ❌ NO | ✅ YES (line 150) | ⚠️ PARTIAL |
| ClarificationAsker | ❌ NO | N/A | N/A | ❌ STATIC |
| ContextExtractor | ❌ NO | N/A | N/A | ✅ OK (doesn't need it) |
| HarmAnalyzer | ❌ NO | N/A | N/A | ✅ OK (doesn't need it) |
| SafetyChecker | ❌ NO | N/A | N/A | ✅ OK (doesn't need it) |

## RECOMMENDATION

### Phase 1: STOP HERE
- Response generation: ✅ Now has adaptive SystemPrompt
- Don't add adaptation to analysis/detection tools (wrong purpose)

### Phase 2: OPTIONAL IMPROVEMENTS (only if needed)
- ClarificationAsker: Could pass style through, make questions formal/casual
- QuestionGenerator: Could move style from UserPrompt to SystemPrompt

### Phase 3: CREATE SHARED CONTEXT (if scaling needed later)
If many new tone-dependent operations are needed:
```go
type UserContextProfile struct {
  Style        string  // formal|casual|playful
  EmotionalTone string  // very_negative|negative|neutral|positive|very_positive
  Topic        string  // work|relationships|family|mental_health|general
  IsFirstMessage bool
  RiskLevel    string  // none|low|medium|high|crisis
}

// Built once per message, passed to tools that need it
```

## CONCLUSION

The adaptive prompt fix (generateConversationalResponse) is **correct and complete for its scope**.
Other tools either:
1. Don't need adaptation (extraction/safety tools)
2. Already get context passed (SuggestionGenerator)
3. Could be improved later if needed (ClarificationAsker, QuestionGenerator)

**Recommendation: Don't refactor. We're NOT overcomplicating - we're only adapting where it matters.**
