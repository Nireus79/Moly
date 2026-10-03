# MOLY - 11-LAYER SYSTEM ARCHITECTURE

**Version**: 1.0  
**Status**: ✅ IMPLEMENTED  
**Date**: Sept 24, 2026

---

## CORE PHILOSOPHY

**"Better asking questions forever than giving one bad piece of advice"**

Moly's orchestrator prioritizes **clarification over blocking**. It never refuses based on incomplete understanding. Denial is the absolute last resort.

---

## KEY ARCHITECTURAL INSIGHTS (Oct 3, 2026)

### 1. Every Message is Data Gold
Each user message provides valuable information that gets:
- **Extracted** (principles, not keywords)
- **Evaluated** (against ethics/safety)
- **Saved** (to build complete context)
- **Analyzed** (maturity updated)

The system never discards data. All messages accumulate into richer context.

### 2. Maturity is THE Control Variable
**Maturity determines system behavior:**
- **Low maturity (< 0.5)**: Ask clarification questions to build context
- **Medium maturity (0.5-0.8)**: Ask prioritized gaps aligned to user's goal
- **High maturity (> 0.8)**: Provide help using complete context

**Critical rule**: Maturity MUST improve with each message. If it doesn't, the system stays in clarification loop.

### 3. Gap Prioritization Drives Natural Conversation
Not all gaps are equal. **Gaps must be prioritized by:**

**Priority 1 (Goal-blocking)** - What's blocking the user's stated goal?
- User goal: "Help me write a first message"
- Real gap: "What's your approach to safety/consent in the message?"
- Not: "How does being dominant inform your approach?" (already explained)

**Priority 2 (Goal-supporting)** - What helps achieve the goal?
- Real gap: "What should the tone be: direct or gradual?"
- Not: "Tell me about your communication style" (generic)

**Priority 3 (Context)** - What's nice-to-know?
- Real gap: "Do you have any previous chats to learn from?"
- Not: "No user profile data" (low impact)

**Priority 4 (Safety)** - Are there concerns?
- Real gap: "Are there any boundaries Christine has mentioned?"
- Always high priority when present

**Key principle**: Follow the user's intent, not the data extraction order.

### 4. Complete Context Around User's Goal
Gap detection, maturity, and response generation must all use:
- **All previous messages** (accumulate data)
- **All extracted entities** (structured knowledge)
- **User's stated goal** (direction)
- **Contact/relationship context** (relevant to goal)

Don't generate gaps from extracted entities. Generate gaps from what's needed to accomplish the user's goal.

---

## THE 11-LAYER ORCHESTRATOR SYSTEM

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

### Layer 3: Context Maturity Assessment (THE KEY CONTROL VARIABLE)
**What happens**: System calculates dynamic context completeness (0-1 score) that MUST IMPROVE with each message:
- User profile completeness (AboutMe fields)
- Relationship context (contacts defined)
- Conversation depth (message history in this conversation)
- **Extracted entities** (data from this message, weighted by confidence)

**Decision Rule - Maturity Controls Behavior**:
- **Maturity < 0.5** (immature): Ask clarification questions → Build context
- **Maturity 0.5-0.8** (gathering): Ask goal-aligned gaps → Deepen understanding
- **Maturity > 0.8** (sufficient): Provide help → Use complete context for goal

**CRITICAL: Maturity Must Improve**
```
Message 1: "Help me write a message"
  Extracted: goal, contact, style (5 entities, confidence=0.87)
  Maturity = 0.3 → Ask clarifying gaps

Message 2: "I focus on safety, consent, respectful communication" 
  Extracted: preferences, values, approach (19 entities, confidence=0.89)
  MATURITY MUST INCREASE → 0.6+
  
✗ IF maturity stays 0.3: System stuck in clarification loop (broken!)
✓ IF maturity increases to 0.6: System re-assesses with new context
```

**Why this matters**:
```
New user: AboutMe=0, Contacts=0, Messages=0, Entities=0
Result: Maturity = 0.0 → Ask what you need help with

After message 2: AboutMe=partial, Contacts=1, Messages=2, Entities=19
Result: Maturity = 0.6+ → Ask goal-specific gaps (not generic ones)

After message 3 (if needed): AboutMe=complete, Contacts=known, Entities=25
Result: Maturity = 0.8+ → Provide targeted help
```

**Implementation Check**:
- [ ] Maturity is calculated after EVERY message
- [ ] Maturity uses ALL accumulated data (not just current message)
- [ ] Gap detection respects maturity threshold
- [ ] System doesn't ask already-answered questions
- [ ] Maturity improves with new extracted data

---

### Layer 4: Context Gap Detection (Goal-Aligned Prioritization)
**What happens**: Moly identifies gaps ALIGNED TO USER'S GOAL, not generic profile gaps:

**WRONG approach** (generates noise):
```
User says: "I want to write a smart first message"
System extracts: "smart", "playful", "dominant", "submissive"
Gap generated: "You mentioned preferring smart and playful. How does this apply?"
← User JUST explained they want to USE these in the message!
```

**CORRECT approach** (natural conversation):
```
User says: "I want to write a smart first message"
Extracted: Goal=write_message, Tone=smart_playful
Real gaps:
1. What's appropriate for first contact with this person?
2. How explicit should you be about interests?
3. What's your safety approach?
← These gaps help achieve the user's goal
```

**Gap Prioritization Rules**:
1. **Identify user's explicit goal** from message (not assumption)
2. **Generate gaps that block/support that goal** (not generic profile gaps)
3. **Rank by impact**: Goal-blocking (HIGH) → Supporting (MEDIUM) → Nice-to-know (LOW)
4. **Ask ONE prioritized gap**, let user answer, then re-assess
5. **Don't ask about extracted data** - use it as context instead

**Example**:
```
User: "I want to write a smart first message to Christine_sub. I have BDSM experience 
       but only real-life. I focus on safety, consent, respectful communication."

Gap analysis:
✗ WRONG: "You said you're experienced. How does this inform your approach?" 
         (Already explained!)
✗ WRONG: "You mentioned preferring respectful. How does this apply?"
         (Explicitly stated how it applies!)
✓ CORRECT: "Based on your approach, what role do you want to take in the opening—
           should you be direct about your interests, or let her respond first?"
         (Helps them write the actual message)

Moly's response: "I love your thoughtful approach to safety and consent. 
One question: In this first message, do you want to be direct about your 
interest in BDSM, or start more subtly and gauge her response?"
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

| Layer | Component | Status | Location | Notes |
|-------|-----------|--------|----------|-------|
| 1 | Context Extraction (no keywords) | ✅ | ContextExtractor (LLM-based) | Extracts contacts, style, intention without hardcoded patterns |
| 2 | Deterministic Principles (Tier 1a/1b) | ✅ | ConstitutionalEvaluator | Tier 1a: hard blocks (no LLM), Tier 1b: signal scan (no LLM) |
| 3 | Context Maturity Assessment | ✅ | main.go + ConstitutionalEvaluator | Maturity < 0.5 defers Tier 2 LLM, always runs Tier 1a |
| 4 | Context Gap Detection | ✅ | ConversationAgent clarification flow | Detects missing info about user/situation |
| 5 | Conflict Detection | ✅ | ConversationAgent + ConflictHandler | Detects inconsistencies, asks "you said X, now Y?" |
| 6-7 | Principle Concern Clarification | ✅ | ConversationAgent.detectPrincipleConcerns() | Detects if clear message involves principle concerns |
| 8 | Socratic Deepening | ✅ | SocraticQuestionSelector | Philosophical questions once context clear |
| 9 | Topic/Contact Change Detection | ✅ | SubjectShiftDetector | Detects conversation pivots |
| 10 | Persistent Questioning | ✅ | ConversationAgent.detectRepeatedConcern() | Deeper questions when user persists after clarification |
| 11 | Denial as Last Resort | ✅ | ConstitutionalEvaluator.ToSafetyAlert() | Only after all layers exhausted |

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

**Status**: All 11 layers fully implemented, tested, and architecturally aligned ✅
**Last Updated**: Sept 25, 2026 - Fixed architectural misalignments (Priority 1-3)
**Verification**: 
- Priority 1 (Critical): Evaluator always called, Tier 1a never skipped ✅
- Priority 2 (Important): Layer 6-7 principle concern detection implemented ✅
- Priority 3 (Robustness): Layer 10 persistent questioning implemented ✅
