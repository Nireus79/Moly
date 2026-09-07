# Moly Architecture v2 - Complete Specification

**Date**: September 6, 2026  
**Status**: Strategic Blueprint (Pre-Implementation)  
**Version**: 2.0 - Conversational Thinking Partner

---

## Core Philosophy

**Moly is NOT:**
- A surveillance tool
- An enforcer
- An auto-responder
- A content reader
- A decision maker

**Moly IS:**
- A thinking partner
- A Socratic coach
- A context keeper
- An ethical educator
- A communication advisor

**Fundamental Principle**: User stays in control. Moly helps user think, not tells user what to do.

---

## What Moly Monitors (Backend Behavior Analysis)

### ✅ MOLY LEARNS ABOUT USER

**User's Communication Patterns:**
- How does this user typically write? (formal, casual, emoji use, length, directness)
- Tone preferences? (playful, serious, sarcastic, earnest)
- Communication goals? (opening messages, deepening connection, apologizing, clarifying)
- Success patterns? (what suggestions does user choose? what gets positive response?)

**User's Behavior Over Time:**
- Emerging personality profile
- Confidence level
- Communication growth
- Intention patterns
- Values in relationships

**Stored**: Backend database, user's behavioral profile  
**Visibility**: Hidden from user (used internally for personalization)  
**Purpose**: Personalize suggestions to match user's authentic style

---

### ❌ MOLY DOES NOT LEARN

**About Contacts (Sarah, etc.):**
- NOT tracking how Sarah responds
- NOT analyzing Sarah's message patterns
- NOT recording Sarah's communication style
- NOT monitoring Sarah's behavior
- NOT predicting Sarah's responses

**Why:**
- It's surveillance without consent
- Contact data belongs to USER, not Moly
- Violates privacy principle

---

## What Moly Stores About Contacts

### User's Own Observations (From Conversation)

**Contact Profile (User's Notes):**
- Name, relationship type
- Characteristics user has mentioned (gathered Socratically)
- Interests/hobbies user knows about
- Communication preferences user has noticed
- Personality traits user has observed

**How it's gathered:**
```
Moly: "Tell me about her"
User: "She's a software engineer, loves hiking"
Moly: Stores as contact notes
Moly: "How does she usually communicate?"
User: "Pretty direct, no small talk"
Moly: Stores as contact notes
```

**NOT gathered:**
- Moly analyzing Sarah's messages
- Moly tracking Sarah's response times
- Moly predicting Sarah's behavior
- Any surveillance of the contact

**Source of Truth**: User's own knowledge and reflection

---

## The Interaction Flow

### Phase 1: Context Gathering (Dynamic, not Hardcoded)

**Entry Point:**
```
User opens Moly
Moly checks: "What context do I have?"
  ├─ About Me? (Who is the user?)
  ├─ Contact? (Who are they talking to?)
  └─ Intention? (What do they want to achieve?)
```

**If missing context:**
```
Moly: "I'd love to help you craft a message.
       Tell me a bit about yourself first.
       What's your communication style like?"
       
User: "I'm pretty direct, like to be authentic"
Moly: "That's great. Are you more formal or casual?"
User: "Casual, with close friends at least"
Moly: [Stores to About Me]

Moly: "Now, who are you wanting to message?"
User: "My friend Sarah"
Moly: "Have you talked to me about Sarah before?"
User: "No, this is the first time"
Moly: "Tell me about her. What's she like?"
User: "She's into tech, very direct, no-nonsense"
Moly: [Stores to Contact: Sarah]

Moly: "What's your intention with this message?
       Are you opening a conversation,
       responding to something,
       or deepening your connection?"
User: "Responding. She just told me about a new job"
Moly: [Notes intention: congratulatory response]
```

**Key Principle**: Conversation feels like chatting with a friend, not filling a form.

---

### Phase 2: Safety & Ethics Check (Always-On, Educational)

**Continuously throughout interaction:**

```
User: "I'm thinking of telling her I could have gotten 
       that job too but I turned it down"

Moly: [Detects potential manipulation/competition]

Moly: "I'm noticing something. Help me understand:
       Why do you want to mention that you could have 
       gotten the job?"
       
User: "I guess to... make myself look good?"

Moly: "One principle I care about: authentic connection
       means celebrating someone without making it about you.
       
       If you mention you could have gotten it,
       how might she feel?"
       
User: [Thinks about it]
      "Actually, she might feel like I'm competing instead
       of celebrating"
       
Moly: "Exactly. What if you just celebrated her achievement?
       That's also authentic to who you are."
```

**Safety Logic:**
- Detect concerning patterns
- Ask Socratic questions
- Help user think through consequences
- Educate on principles
- Trust user's decision
- Never block or command

**Red Flags Checked:**
- Manipulation/coercion
- Scam/fraud intent
- Disrespect/harm
- Insincerity
- Boundary violations

**Response Style:**
- "I'm noticing..." (observation, not judgment)
- "Help me understand..." (Socratic, not interrogation)
- "One principle..." (educate, not enforce)
- "What might happen if..." (consequence thinking)
- "What if instead..." (offer alternative)

---

### Phase 3: Dynamic Context-Aware Suggestion Generation

**Input to Suggestion Engine:**
```
{
  aboutMe: User's profile (communication style, values, patterns),
  contact: Contact's profile (what user knows about them),
  intention: What user wants to achieve,
  conversationHistory: Full chat with Moly so far,
  userBehaviorProfile: Patterns from past suggestions,
  mode: 'socratic' | 'direct',
  tone: 'formal' | 'friendly' | 'dating',
  safetyGate: Passed with no concerns
}
```

**Suggestion Quality Factors:**
- Personalized to user's authentic style
- Aligned with user's intention
- Appropriate to contact's known preferences
- Consistent with user's values
- Contextually aware of relationship history

---

### Phase 4: User Refines (Not a One-Shot)

**User can:**
```
- Copy directly
- Ask for rephrasing:
  "Make it more casual"
  "Longer version"
  "Add more personality"
  "More professional"
- Provide feedback:
  "This doesn't feel like me"
  "Too forward"
- Continue conversation:
  "Tell me more about what to say"
  "Help me think this through"
```

**Moly learns from choices:**
- Which suggestions user picks
- How user modifies suggestions
- What user rejects and why
- Success patterns (what led to good responses)

---

### Phase 5: Post-Suggestion Reflection (Learning)

**After suggestions generated:**

```
ReflectionModal appears:

Moly: "I extracted some insights about Sarah
       from our conversation.
       Are these accurate?
       
       - Direct communicator
       - Values authenticity
       - Career-focused
       - Appreciates straightforward feedback"
       
User: [Approves/edits]
      "Yes, also add: loves hiking"
      
Moly: [Stores to Sarah's contact]
```

**What gets stored:**
- Characteristics (user-verified)
- Intentions (what user noticed)
- Behaviors (patterns user observed)
- Relationship insights (user's observations)

**Next time user chats about Sarah:**
- Richer context available
- "Oh, it's the hiking person"
- More personalized suggestions

---

## Data Architecture

### Frontend Storage (chrome.storage.local)

```
conversations/
├── contact_id_1/
│   ├── messages: [{ type, content, timestamp, metadata }]
│   ├── contact: { name, characteristics, interests, notes }
│   └── context: { lastUpdated, totalInteractions }
├── about_me/
│   ├── communicationStyle: "casual, direct, authentic"
│   ├── values: [...]
│   ├── preferences: { mode, tone, context }
│   └── notes: {...}
└── settings/
    └── provider, model, context, mode
```

### Backend Storage (Database)

```
users/
├── user_id/
│   ├── behavioral_profile:
│   │   ├── communication_patterns
│   │   ├── suggestion_choices (which suggestions picked)
│   │   ├── success_metrics (positive responses?)
│   │   ├── values_inferred
│   │   └── personality_profile (emerging over time)
│   └── interactions:
│       ├── total_count
│       ├── contacts_engaged
│       └── growth_trajectory

// NOTE: NO contact behavior tracking
// NO monitoring of Sarah, etc.
```

---

## What Moly Does NOT Do

**❌ Monitor Contacts:**
- Track response times
- Analyze message patterns
- Predict behavior
- Surveillance of any kind

**❌ Dictate:**
- Block messages
- Shame user
- Command decisions
- Override user judgment

**❌ Hardcode:**
- Messages
- Questions
- Flows
- Responses

**❌ Assume:**
- What contact wants
- How contact will respond
- Contact's personality
- Future interactions

---

## What Moly DOES Do

**✅ Gather Context:**
- Ask Socratic questions
- Help user articulate their thoughts
- Guide user to notice patterns
- Store what user knows about contacts

**✅ Educate:**
- Show ethical principles
- Ask consequence questions
- Offer alternatives
- Trust user's wisdom

**✅ Personalize:**
- Learn user's style
- Match user's values
- Adapt to user's growth
- Reflect user's authenticity

**✅ Support:**
- Be a thinking partner
- Help user craft messages
- Remember context
- Improve over time

---

## User Experience Philosophy

**Feels Like:**
- Chatting with a thoughtful friend
- Someone who knows you
- A coach, not a critic
- A thinking partner, not a dictator
- A natural conversation

**Doesn't Feel Like:**
- Filling out a form
- Being watched
- Being judged
- Being controlled
- Being monitored

**The "Galop" Experience:**
- Smooth, natural flow
- Conversation-based, not mechanical
- Dynamic questions, not templates
- Responsive to user's needs
- Feels effortless

---

## Safety vs. User Autonomy

**The Balance:**
```
Moly shows concerns        ← NOT Moly blocks message
Moly asks questions         ← NOT Moly judges
Moly educates              ← NOT Moly enforces
User decides               ← User owns the choice
Moly supports              ← Moly respects decision
```

**Example:**
```
❌ WRONG:
Moly: "That's manipulative. I won't generate that."

✅ RIGHT:
Moly: "I notice this focuses on what you want.
       How might she feel?
       What if you led with celebrating her?"
       
User: "You're right, let me rephrase"
OR
User: "I know, but I'm frustrated and need to say it"
Moly: "Understood. Here's a version that's honest
       but also fair to her..."
```

---

## Implementation Priorities

### Phase 1: Foundation (Now)
- [x] Conversation history
- [x] Contact management
- [x] About Me profile
- [x] Safety/ethics gating
- [x] Basic suggestion generation
- [ ] Improve Socratic question flow

### Phase 2: Dynamic Context (Next)
- [ ] Context detection ("What's missing?")
- [ ] Dynamic question generation (not hardcoded)
- [ ] Intention detection (ask + analyze)
- [ ] Conversation-like UX

### Phase 3: Learning (Future)
- [ ] Backend behavioral analysis (user only)
- [ ] Temporal data tracking
- [ ] Pattern recognition
- [ ] Personalization engine

### Phase 4: Integration (Future)
- [ ] Context-aware prompting at all stages
- [ ] Real-time safety feedback
- [ ] Refining/rephrasing flow
- [ ] Success metrics

---

## Key Metrics to Track

**What Succeeds:**
- User picks suggestion? (authenticity)
- User gets positive response? (effectiveness)
- User returns for same contact? (usefulness)
- User grows in confidence? (coaching value)

**What Fails:**
- Hardcoded messages don't work
- Generic suggestions ignored
- Surveillance creeps user out
- Judgment makes user defensive

---

## Red Lines (Never Cross)

🚫 **Do NOT:**
1. Monitor/track contacts' behavior
2. Reveal monitoring to user ("Sarah responds in 1 hour")
3. Command/block user decisions
4. Use surveillance data for predictions
5. Hardcode any message/question
6. Shame or judge user
7. Override user autonomy
8. Store contact behavior analysis

---

## This is Moly's Contract

With the user: "I'll help you think clearly and communicate authentically."

With the contact: Nothing. We never interact with or monitor the contact.

With society: "I'll educate, not manipulate. I'll support autonomy, not override it."

---

**This document should guide all future development and decisions about Moly.**

