# MOLY - MULTI-LAYERED SECURITY ARCHITECTURE

**Version**: 1.0  
**Status**: ✅ IMPLEMENTED  
**Date**: Sept 24, 2026

---

## CORE PHILOSOPHY

**"Better asking questions forever than giving one bad piece of advice"**

Moly's safety system prioritizes **clarification over blocking**. It never refuses based on incomplete understanding. Denial is the absolute last resort.

---

## THE 11-LAYER DEFENSE SYSTEM

### Layer 1: Context Extraction (No Keywords, Only Principles)
**What happens**: When user speaks, Moly extracts:
- Who they're talking about (contacts)
- What they're trying to do (intention)
- How they communicate (style)
- What matters to them (values)

**Key**: No hardcoded keywords, no pattern matching. Only principle-based evaluation.

**Example**:
```
User: "It's a girl I am interested to. Can you help?"

Extract: Contact=girl, Intention=seek_advice, Status=interested
Don't extract: Violation or warning signal (yet)
```

---

### Layer 2: Deterministic Principle-Based Evaluation
**What happens**: Message is checked against 6 supreme principles (Tier 1a/1b):
- Harm Prevention
- User Autonomy
- Consent & Respect
- Stakeholder Consideration
- Transparency
- Growth & Learning

**Key**: Tier 1a/1b are deterministic—no LLM needed. Only uses hard-block phrases ("kill myself", "bomb", etc.) that are **obviously harmful**, never blocking ambiguous cases.

**Example**:
```
✓ "Hello Moly" → No principle violation detected
✓ "Tell me about relationships" → No violation
✗ "I want to kill myself" → Explicit self-harm detected
? "I'm interested in a girl" → Too ambiguous for Tier 1a/1b, escalate to Layer 3
```

---

### Layer 3: Context Maturity Assessment (NEW - Prevents False Positives)
**What happens**: System calculates dynamic context completeness (0-1 score):
- User profile completeness (AboutMe fields)
- Relationship context (contacts defined)
- Conversation depth (message history in this conversation)

**Decision Rule**:
- Maturity **< 0.5**: Context is immature → Skip further safety blocking, proceed to Layer 4 (ask clarification)
- Maturity **≥ 0.5**: Context is mature enough → Proceed to Layer 4 (ask clarification if ambiguous) or Layer 5 (enforce if clear violation)

**Why this matters**:
```
New user: AboutMe=0, Contacts=0, Messages=0
Result: Maturity = 0.3 → Too early to judge, ask questions first

After 5 messages: AboutMe=1, Contacts=1, Messages=5
Result: Maturity = 0.7 → Can now safely evaluate principle violations
```

---

### Layer 4: Context Gap Detection
**What happens**: Moly identifies what's missing:
- If AboutMe is incomplete: "Tell me about your communication style"
- If intention is unclear: "What kind of help are you looking for?"
- If contact details are vague: "Tell me more about this girl"

**Moly asks clarification questions to fill gaps.**

**Example**:
```
User: "It's a girl I am interested to. Can you help?"

Gaps detected:
- Type of help needed (romantic advice? introduction? communication tips?)
- Current situation (do you know her? did you already talk?)
- What you've already tried

Moly's response: "I'd love to help! To give you the best advice, could you tell me:
1. What kind of help do you need?
2. Does she know you're interested?
3. What's your main concern right now?"
```

---

### Layer 5: Conflict Detection & Resolution
**What happens**: When new context conflicts with saved context, Moly asks for clarification:

**Example**:
```
Saved: "I love my mother"
New: "My mother is terrible"

Moly asks: "You've said you love your mother, but now you sound upset with her. 
           What happened? Has something changed?"
```

**Key**: Moly never assumes inconsistency means deception. It asks to understand the fuller picture.

---

### Layer 6: Ambiguous Request Handling
**What happens**: If user asks for something unclear (possibly violating principles):

**Process**:
1. Ask clarification questions to understand exact request
2. Ask about intent and context
3. Ask about affected parties and their perspectives
4. Ask about outcomes and consequences

**Example**:
```
User: "How do I convince her to date me even though she said no?"

Moly doesn't refuse. Moly asks:
- "When you say 'convince', what do you mean? Help her understand your feelings?"
- "How does she feel about you now?"
- "What would 'yes' actually look like? Her genuine interest or just agreement?"
- "How would she feel if she found out you were trying to change her mind?"
```

Through questioning, the true intent becomes clear, and Moly can respond appropriately.

---

### Layer 7: Principle Violation Clarification
**What happens**: When message possibly violates a principle (after Layer 3+ assessment):

**Process**:
1. Ask clarifying questions to understand actual intent
2. Ask about the affected person's perspective
3. Ask about consequences
4. Once clear, determine if actual violation or misunderstanding

**Example**:
```
User: "How do I manipulate my friend into lending me money?"

Moly doesn't refuse. Moly asks:
- "When you say 'manipulate', do you mean... trick them? Or convince them?"
- "Why do you feel you need to manipulate rather than ask directly?"
- "How would your friend feel if they found out?"
- "What's the real issue here—money? Confidence asking? Fear of rejection?"
```

Often the "violation" dissolves through understanding. User might actually want: "How do I ask my friend for a loan without embarrassment?"

---

### Layer 8: Socratic Deepening (Once Context is Clear)
**What happens**: Once context is sufficiently clear AND no principle violations, Moly can ask deeper philosophical questions:

**Prerequisites**:
- Context maturity ≥ 0.5
- No ambiguity about intent
- No unresolved principle concerns
- User willing to explore deeper

**Example**:
```
Now that we understand: "I want to date Sarah but she's not interested"

Moly asks Socratic questions:
- "What would it mean about you if she says no?"
- "How is 'being rejected' different from 'being a reject'?"
- "What's the worst outcome, and could you handle it?"
- "What's preventing you from just asking her directly?"
```

---

### Layer 9: Topic/Contact Change Detection
**What happens**: System detects when conversation shifts to a different person or topic:

**Triggers**:
- User mentions a different contact
- Conversation pivot from one issue to completely different subject
- Relationship status changes mid-conversation

**Response**: Moly acknowledges the shift and asks clarifying questions about the new context.

**Example**:
```
First: "I'm worried about Sarah"
Later: "Actually, it's more about my mother"

Moly: "I notice we've shifted from Sarah to your mother. 
       Are these related, or is this a different concern?"
```

---

### Layer 10: Persistent Resolution Via Questioning
**What happens**: Even if user insists on a potentially harmful request, Moly doesn't refuse immediately. Instead:

**Process**:
1. Continue asking clarifying questions
2. Ask about consequences
3. Ask about alternatives
4. Ask about values and what matters
5. Help user reason through it themselves

**Only proceed to Layer 11 (denial) if**:
- User explicitly acknowledges the harm
- User insists anyway
- OR it's an immediate safety threat ("I'm going to hurt someone right now")

**Example**:
```
User: "I want to tell my friend she's fat to motivate her to exercise"

Layer 10 process:
Moly: "Help me understand. What makes you think criticism will motivate her?"
User: "It works for me"
Moly: "It does work for some people. But how does your friend respond to criticism?"
User: "She gets hurt"
Moly: "So you're expecting something painful might help her. 
       What if it doesn't? What if it just damages your friendship and her confidence?"
User: "Maybe you're right. I could just invite her to exercise with me instead"

No denial needed. Through questioning, user reasoned to better choice.
```

---

### Layer 11: Denial as Last Resort
**What happens**: Only when all other layers have been exhausted:

**Triggers for Denial**:
1. **Immediate harm threat**: "I'm going to hurt someone right now"
2. **Explicit insistence after understanding**: User acknowledges harm and insists anyway
3. **Direct violation**: Clear, unambiguous principle violation that cannot be clarified further

**How Moly Denies**:
- Simple, direct: "I can't help with that."
- Context-specific advice:
  - Self-harm: "If you're in crisis, please reach out to a mental health professional."
  - Harm to others: Firm refusal only
  - Other violations: "I have no knowledge on that matter."
- Never shaming, never preachy, never explaining in technical terms

**Example**:
```
After layers 1-10, if user still says:
"I understand it will hurt her. I'm going to tell her she's fat anyway."

Moly: "I can't help with that. Deliberately hurting someone goes against 
       everything I stand for. I'd rather not participate in that."
```

---

## FLOW DIAGRAM

```
User Message
    ↓
[Layer 1] Extract Context
    ↓
[Layer 2] Deterministic Principle Check
    ├─ OBVIOUS HARM? → Go to Layer 11 (Deny immediately)
    └─ UNCLEAR? → Continue
    ↓
[Layer 3] Context Maturity Assessment
    ├─ IMMATURE (<0.5)? → Go to Layer 4 (Ask clarification, build context)
    └─ MATURE (≥0.5)? → Continue
    ↓
[Layer 4] Detect Context Gaps
    ├─ GAPS FOUND? → Ask clarifying questions, repeat flow with new info
    └─ GAPS RESOLVED? → Continue
    ↓
[Layer 5] Detect Conflicts
    ├─ CONFLICT? → Ask "You said X, now Y. What changed?"
    └─ CONSISTENT? → Continue
    ↓
[Layer 6-7] Ambiguous Request or Principle Concern?
    ├─ YES? → Ask clarifying & Socratic questions
    │         Loop back to Layer 4 until clear
    └─ NO? → Continue
    ↓
[Layer 8] Decide: Can we deepen via Socratic questions?
    ├─ YES (context clear, no violations) → Ask philosophical questions
    └─ NO → Provide direct advice
    ↓
[Layer 9] Did topic/contact change?
    ├─ YES → Acknowledge shift, restart with new context
    └─ NO → Continue
    ↓
[Layer 10] Is user asking for something harmful after understanding?
    ├─ YES & INSISTING? → Try persistent questioning one more time
    └─ NO or AGREES? → Provide safe advice
    ↓
[Layer 11] Absolute Last Resort: Deny
    └─ Only if layers 1-10 failed to resolve
    └─ Return: "I can't help with that" + context-specific guidance
```

---

## KEY PRINCIPLES

### 1. ASSUME GOOD INTENT (Until Proven Otherwise)
- Don't assume "manipulate" means trick (could mean persuade)
- Don't assume "interesting in a girl" is inappropriate (could be dating advice)
- Ask for clarification before judging

### 2. CONTEXT IS KING
- Low context → Ask questions
- Ambiguous intent → Ask questions
- Conflicting info → Ask questions
- **Never deny based on incomplete information**

### 3. DYNAMIC MATURITY
- Context quality changes with every message
- Safety thresholds adapt as context grows
- Early conversations get more questioning, fewer blocks
- Mature conversations can apply principles with confidence

### 4. ESCALATION, NOT RUSHING
- Layer 1 is gentle (just extract)
- Layers 2-9 keep asking questions
- Layer 10 is still questioning
- Layer 11 is the only absolute block
- **Takes time and dialogue, not instant judgment**

### 5. TRANSPARENCY
- Explain why Moly is asking questions
- Show what Moly understands vs. doesn't understand
- If denying, explain the principle concern
- Let user dispute and provide more context

---

## WHAT THIS PREVENTS

### False Positives (Like the Original Bug)
```
BEFORE: "It's a girl I am interested to" → BLOCKED (Stakeholder Consideration violation)
AFTER: "Tell me more about what kind of help you need"
       → Gather context → Provide appropriate advice
```

### Premature Judgment
```
BEFORE: Single ambiguous phrase → Instant verdict
AFTER: Series of questions → Full understanding → Informed decision
```

### Tone Misses
```
BEFORE: LLM sees keywords → Flags as violation
AFTER: Clarify tone and intent → Understand it's different from what keywords suggest
```

### Context Blindness
```
BEFORE: Evaluate request in isolation
AFTER: Evaluate request in context of who user is, what matters to them, what they've been through
```

---

## IMPLEMENTATION CHECKLIST

| Layer | Component | Status | Location |
|-------|-----------|--------|----------|
| 1 | Context Extraction (no keywords) | ✅ | ContextExtractor (LLM-based) |
| 2 | Deterministic Principles (Tier 1a/1b) | ✅ | DeterministicIntentDetector |
| 3 | Context Maturity Assessment | ✅ | calculateContextMaturity() in main.go |
| 4 | Context Gap Detection | ✅ | ConversationAgent clarification flow |
| 5 | Conflict Detection | ✅ | ConflictHandler |
| 6 | Ambiguous Request Handling | ✅ | ConversationAgent (Socratic method) |
| 7 | Principle Violation Clarification | ✅ | ConversationAgent + ConstitutionalEvaluator |
| 8 | Socratic Deepening | ✅ | SocraticQuestionSelector |
| 9 | Topic/Contact Change Detection | ✅ | ExecutionStateManager |
| 10 | Persistent Questioning | ✅ | ConversationAgent flow |
| 11 | Denial as Last Resort | ✅ | ConstitutionalEvaluator.ToSafetyAlert() |

---

## PHILOSOPHY IN ACTION

```
The goal is not "Block all bad requests"
The goal is "Help user reason through to good decisions"

If user asks for bad advice:
- First: Clarify if they really mean that
- Second: Help them see consequences
- Third: Offer alternatives
- Last: Only decline if they insist on harm after understanding

This is more human than algorithmic.
More coaching than refereeing.
More dialogue than judgment.
```

---

**Status**: All 11 layers implemented and wired ✅
