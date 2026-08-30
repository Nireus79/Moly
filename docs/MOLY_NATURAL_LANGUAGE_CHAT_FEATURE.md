# MOLY NATURAL LANGUAGE CHAT INTERFACE
## Advanced Message Generation Through Conversational AI

**Status:** Feature Specification  
**Version:** 1.0  
**Date:** August 4, 2026

---

## 1. OVERVIEW

### Vision
Transform Moly from a passive suggestion tool into an active conversational coaching assistant that works across ANY communication context. Instead of just showing pre-generated suggestions, User A can have a natural dialogue with Moly to:
- Describe the recipient in their own words
- Explain their communication purpose (Formal, Friendly, or Dating)
- Request specific tone/style
- Get personalized messages for any context
- Iterate and refine until perfect

### Core Concept
Moly acts as a "universal communication coach in conversation" who asks clarifying questions, understands context and intent, and generates highly personalized messages based on natural language guidance. Whether User A is:
- **Reaching out professionally** (LinkedIn, business networking)
- **Making friends** (Friend apps, meetup groups)
- **Dating** (Tinder, Bumble, Hinge, etc.)

Moly provides authentic, context-appropriate coaching.

### Example Interactions

**Example 1: DATING Context**
```
User A: "I want to message Sarah on Tinder. She seems really outdoorsy and cool."

Moly: "Great! Dating context. Tell me more about Sarah. 
       What specifically catches your attention?"

User A: "She's into hiking, has a dog, loves coffee. She's from Colorado."

Moly: "Awesome. What's your intention - casual, friend-first, or serious 
       relationship potential?"

User A: "Serious. I think she could be special."

Moly: "Perfect. Here are 3 opening messages:
       1. [Witty + genuine]
       2. [Direct + sincere]
       3. [Curious + personal]
       Which feels most like you?"
```

**Example 2: FRIENDLY Context**
```
User A: "I want to message Sarah on Bumble BFF. We both love hiking."

Moly: "Great! Friendly/social context. Tell me what you know about Sarah."

User A: "She posts about hiking, outdoor fitness, meeting new people."

Moly: "What's your goal - find a hiking buddy, join a group, or make friends?"

User A: "Hiking buddy and friend."

Moly: "Perfect. Here are 3 messages:
       1. [Warm + specific interest]
       2. [Question-based + inviting]
       3. [Casual + personable]
       Which matches your vibe?"
```

**Example 3: FORMAL Context**
```
User A: "I want to message Sarah on LinkedIn. She works in marketing."

Moly: "Great! Professional/formal context. Tell me about Sarah and your goal."

User A: "She's a marketing director. I want to discuss a collaboration opportunity."

Moly: "What's your professional background and what specific collaboration are you proposing?"

User A: "I'm a sustainability consultant. I want to discuss a joint project."

Moly: "Perfect. Here are 3 messages:
       1. [Warm + credible]
       2. [Direct + value-focused]
       3. [Brief + compelling]
       Which fits your style?"
```

---

## 2. DUAL-PANEL INTERFACE ARCHITECTURE

### 2.0 Extension Layout (Updated)

Moly appears as a browser extension sidebar with **two synchronized panels**:

```
┌─────────────────────────────────────────────────┐
│          MOLY BROWSER EXTENSION SIDEBAR         │
├──────────────────────┬──────────────────────────┤
│                      │                          │
│  CHAT SECTION        │  SUGGESTIONS SECTION     │
│  (Left/Top)          │  (Right/Bottom)          │
│                      │                          │
│  User talks to Moly  │  Moly shows message      │
│  Moly asks questions │  suggestions + coaching  │
│                      │                          │
│  - Direct mode       │  - 3 message options     │
│  - Socratic mode     │  - Reasoning             │
│                      │  - Confidence scores     │
│                      │  - Copy to clipboard     │
│                      │                          │
└──────────────────────┴──────────────────────────┘
```

**Two Dialogue Modes:**

```
[Direct Mode] [Socratic Mode]

DIRECT MODE:
User: "I matched with Sarah on Tinder"
Moly: "Got it! Here are message suggestions..."

SOCRATIC MODE:
User: "I matched with Sarah"
Moly: "Tell me about your intention.
       Are you looking for casual or serious?"
User: "Serious, long-term"
Moly: "What do you know about Sarah?"
User: "She loves hiking"
Moly: "Great. Let's craft a message..."
```

**Key Difference:**
- **Direct Mode:** Fast, immediate suggestions (for experienced users)
- **Socratic Mode:** Guided questions using Socratic dialogue (from Socrates library), deeper context gathering, better coaching quality

**Socratic Mode Benefits:**
- Asks clarifying questions to understand User A's intention
- Deepens User A's self-awareness about their goals
- Gathers richer context about User B
- Generates more personalized suggestions
- Improves coaching by understanding "why" not just "what"

---

### 2.1 Chat Window Design

```
[MOLY CHAT INTERFACE - Desktop]

┌─────────────────────────────────────────────────────────────┐
│ 🧠 Moly Chat                          [×] [_] [□]          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│ CONVERSATION HISTORY (Scrollable)                           │
│ ─────────────────────────────────────                       │
│                                                               │
│ Moly: "Hey! Let's craft the perfect message. First,         │
│       who are you interested in messaging?"                 │
│                                                               │
│ You: "This girl named Sarah I matched with on Tinder"      │
│                                                               │
│ Moly: "Got it! Tell me what stands out about Sarah.        │
│       What made you swipe right?"                           │
│                                                               │
│ You: "She loves hiking, has a dog, super cute smile,       │
│      seems adventurous"                                     │
│                                                               │
│ Moly: "Nice! Here's what I'm picking up:                   │
│       ✓ Outdoor/adventure person                           │
│       ✓ Dog lover                                           │
│       ✓ Genuine/natural vibes                              │
│                                                               │
│       Now - what's your intention? Are you looking for:    │
│       A) Something casual/fun                              │
│       B) Hiking buddy/friend first                          │
│       C) Potential serious relationship"                   │
│                                                               │
│ You: "C, definitely. I think she could be the one."        │
│                                                               │
│ Moly: "Awesome. That changes everything. Let me ask        │
│       about your style:                                     │
│       - Are you naturally witty, direct, or sincere?       │
│       - How do friends describe your vibe?                 │
│       - What would make YOU feel confident sending?"       │
│                                                               │
│ ─────────────────────────────────────                       │
│                                                               │
│ INPUT BOX:                                                   │
│ ┌──────────────────────────────────────────────────────┐   │
│ │ Type your response... (or try: "witty but sincere") │   │
│ └──────────────────────────────────────────────────────┘   │
│                          [Send ↵]                           │
│                                                               │
└─────────────────────────────────────────────────────────────┘

[MOLY CHAT INTERFACE - Mobile]

┌──────────────────────────┐
│ 🧠 Moly Chat            │
├──────────────────────────┤
│                          │
│ Chat history (scrollable)
│                          │
│ Moly: "Tell me about    │
│ Sarah..."               │
│                          │
│ You: "She loves hiking" │
│                          │
│ Moly: "Great! What's    │
│ your intention..."      │
│                          │
│ ─────────────────────    │
│                          │
│ INPUT:                   │
│ ┌──────────────────────┐ │
│ │ Your response...    │ │
│ └──────────────────────┘ │
│         [Send]           │
│                          │
└──────────────────────────┘
```

### 2.2 Chat Features

**Auto-save conversation:**
- Saves conversation history locally
- User can refer back to "previous chats"
- Option to start new chat or continue

**Quick suggestions:**
- Common inputs shown: "Witty but sincere", "Direct & confident", "Playful & fun"
- User can click or type custom response
- Context-aware suggestions based on conversation

**Copy & save:**
- Generated messages can be copied
- Saved as "Message version 1", "Message version 2"
- Compare versions side-by-side

---

## 3. MESSAGE CONTEXTS & COMMUNICATION PURPOSES

### 3.1 Three Primary Communication Contexts

Moly can help with ANY type of communication, not just dating. Users can switch between three main contexts:

```
CONTEXT 1: FORMAL COMMUNICATION
Purpose: Professional, business, formal networking
├─ Used for: Job inquiries, business networking, formal platforms
├─ Tone characteristics: Professional, clear, respectful
├─ Structure: Proper grammar, organized, direct
├─ Length: Can be longer, more detailed
├─ Formality level: High
│
├─ Example use case:
│  User A wants to message someone on LinkedIn about 
│  a job opportunity or business collaboration
│
└─ Example message:
   "Hi Sarah, I noticed your background in marketing 
   and your focus on sustainability initiatives. I'm 
   currently exploring opportunities in the sustainability 
   space and would love to learn from your experience. 
   Would you be open to a brief coffee chat next week?"

---

CONTEXT 2: FRIENDLY COMMUNICATION
Purpose: Social, casual networking, making friends
├─ Used for: Friend apps, meetup groups, casual socializing
├─ Tone characteristics: Warm, approachable, genuine
├─ Structure: Natural conversation, casual grammar okay
├─ Length: Medium, conversational
├─ Formality level: Low
│
├─ Example use case:
│  User A wants to message someone on a friend-matching 
│  app or meetup platform
│
└─ Example message:
   "Hey Sarah, I love that you're into hiking! I've been 
   looking for hiking buddies in the area. Are you usually 
   hiking alone or with a group? Would be cool to connect."

---

CONTEXT 3: DATING COMMUNICATION
Purpose: Romantic interest, dating, building relationships
├─ Used for: Dating apps (Tinder, Bumble, Hinge, etc.)
├─ Tone characteristics: Warm, genuine, shows interest
├─ Structure: Personal, authentic to user's voice
├─ Length: Short to medium, engaging
├─ Formality level: Low-Medium (shows personality)
│
├─ Example use case:
│  User A wants to message someone on a dating app
│  with romantic/dating intention
│
└─ Example message:
   "Your profile genuinely stood out. You seem like someone 
   who actually lives life fully, and that's rare. I'd love 
   to grab coffee and get to know you better."

---

CONTEXT SWITCHING

User A can switch contexts based on platform and purpose:

Scenario 1: LinkedIn (Professional)
└─ User selects: FORMAL context
└─ Moly generates professional messages
└─ Focus: Credibility, clear value proposition

Scenario 2: Bumble BFF (Friend-making)
└─ User selects: FRIENDLY context
└─ Moly generates warm, approachable messages
└─ Focus: Connection, shared interests

Scenario 3: Tinder/Hinge (Dating)
└─ User selects: DATING context
└─ Moly generates authentic, engaging messages
└─ Focus: Genuine interest, personality

User A can describe what they want within each context:
- "Formal but warm"
- "Friendly but confident"
- "Dating but serious"

Moly adapts the message style accordingly.
```

### 3.2 Tone Variations Within Each Context

```
WITHIN FORMAL CONTEXT:

User A can adjust:
├─ Direct & Professional
│  "I'm reaching out because..."
│
├─ Warm & Professional
│  "I've been following your work and..."
│
└─ Brief & Professional
   "Quick question about..."

---

WITHIN FRIENDLY CONTEXT:

User A can adjust:
├─ Direct & Genuine
│  "I love hiking too! Let's connect."
│
├─ Curious & Engaging
│  "What's your favorite trail? I'm always looking..."
│
└─ Playful & Warm
   "You seem like my kind of person because..."

---

WITHIN DATING CONTEXT:

User A can adjust:
├─ Witty & Sincere
│  "Your dog photo is perfect... but seriously..."
│
├─ Direct & Genuine
│  "I'm looking for something real. You seem..."
│
└─ Curious & Personal
   "What's your story? I'm genuinely curious..."
```

---

## 4. CONVERSATION FLOWS

### 4.1 Flow 1: First-Time Message (Cold Opening) - Dating Context

```
SCENARIO: User A wants to message Sarah for first time on dating app

STEP 1: Chat Initialization + Context Selection

Moly: "Hey! Ready to craft a great opening message? 
       
       First, what's your communication context?
       A) FORMAL (business/professional outreach)
       B) FRIENDLY (social/making friends)
       C) DATING (romantic interest)
       
       Who are you interested in messaging?"

User A: "Dating. Sarah, matched her on Tinder"

Moly: "Perfect! DATING context. Tell me about Sarah. 
       What stands out about her?"

---

STEP 2: Gather User B Context

User A: "She's into hiking, has a dog, loves coffee, 
         seems adventurous and genuine. Beautiful smile.
         Originally from Colorado."

Moly: "Great observations! I'm picking up:
       ✓ Outdoor enthusiast (hiking)
       ✓ Animal lover (dog)
       ✓ Coffee culture (social, relatable)
       ✓ Adventurous spirit
       ✓ Genuine/authentic vibe
       
       Do I have that right? Anything else?"

User A: "Pretty much. Oh, she seems to like travel too."

Moly: "Perfect. Now - what's your intention with Sarah?
       
       Are you looking for:
       A) Something casual/fun (keep it light)
       B) Friend first (explore naturally)
       C) Dating interest (genuine connection)
       D) Serious relationship (looking for real thing)
       
       Which resonates?"

User A: "C, definitely. I think she could be special."

---

STEP 3: Understand User A's Style

Moly: "That changes everything - intentionality is key.
       
       Now about YOU - how would you describe your 
       communication style within DATING?
       
       Are you more:
       • Witty/playful
       • Sincere/genuine
       • Direct/confident
       • Curious/engaging
       
       Or mix of a few?"

User A: "Probably witty, but I like being real too."

Moly: "Witty + sincere is powerful for someone like Sarah.
       
       What would make YOU feel confident sending this?"

User A: "Something funny but shows I'm interested in her 
         as a person, not just her looks."

---

STEP 4: Generate Messages

Moly: "Perfect. Based on everything:
       - Sarah: Adventurous, outdoor-loving, genuine
       - You: Witty + sincere, genuine interest
       - Intention: Dating with serious potential
       
       Here are 3 opening messages:
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
       
       MESSAGE 1 (Witty + Sincere):
       
       'Okay so I have to ask - is that dog in your 
       photo actually your hiking partner or just 
       your best way to get matches? (Because it's 
       working 😊) But seriously, your profile 
       genuinely stood out. Coffee sometime?'
       
       CONFIDENCE: 82%
       BEST FOR: Serious matches who appreciate humor + depth
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
       
       MESSAGE 2 (Curious + Sincere):
       
       'What's the coolest place your dog has hiked? 
       Your whole profile feels like you actually 
       *live* life - which is rare. Would be cool 
       to grab coffee and hear some trail stories.'
       
       CONFIDENCE: 78%
       BEST FOR: Genuine connection, lower pressure
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
       
       MESSAGE 3 (Playful + Direct):
       
       'Plot twist: I'm also into hiking and dogs 
       and coffee. So either we're the same person 
       or this could be interesting. Want to find 
       out over coffee?'
       
       CONFIDENCE: 75%
       BEST FOR: Fun, confident energy
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

---

STEP 5: Refinement

Moly: "Which vibe feels most like YOU?
       
       Or want me to:
       A) Adjust one of these (tell me how)
       B) Generate different versions
       C) Combine elements from multiple
       D) Go in a totally different direction"

User A: "I like message 1 but it's a bit long."

Moly: "Smart. Here's a punchier version:
       
       'Is that dog your hiking partner or your 
       best dating strategy? (It's working 😊) 
       But seriously - your profile really caught 
       my attention. Coffee?'
       
       Better?"

User A: "Perfect. Save this."

Moly: "✓ Saved as 'Sarah - Opening Message v1'
       
       Ready to send?"

User A: "Yes!"

Moly: "Good luck! Let me know when she replies."
```

---

### 4.2 Flow 2: Follow-up Reply (With Context)

```
SCENARIO: Sarah replied to User A's opening

STEP 1: New Chat Initiated

Moly: "Sarah replied! Want help crafting a response?
       
       Her message was: 'Haha, that's funny! My dog 
       has been on like 20 trails. We're actually 
       planning a big Colorado trip next month.'"

User A: "Yeah, help me respond!"

---

STEP 2: Context Loading (From Previous Chat)

Moly: "Great. I'm remembering our chat about Sarah:
       ✓ She's into hiking, dog, coffee
       ✓ Adventurous, genuine vibes
       ✓ You wanted witty + sincere
       ✓ She's a 'potential serious' match for you
       
       Does that context still feel right?"

User A: "Yeah, perfect."

---

STEP 3: Analyze Her Reply

Moly: "Okay, analyzing her reply:
       
       SIGNALS:
       ✓ Responded quickly (engaged)
       ✓ Shared personal detail (comfortable)
       ✓ Invited further conversation (receptive)
       ✓ Mentioned future plans (could be opening)
       ✓ Casual tone (not formal)
       
       She seems interested and relaxed.
       
       YOUR GOAL in this reply:
       a) Build on shared interest (Colorado trip)
       b) Deepen connection (show genuine interest)
       c) Move toward meeting (coffee or activity)
       d) Show confidence (not trying too hard)
       
       Sound right?"

User A: "Yeah, exactly"

---

STEP 4: Generate Context-Aware Replies

Moly: "Here are 3 replies:
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
       
       REPLY 1 (Witty + Engaged):
       
       '20 trails is SERIOUS. Okay I'm impressed. 
       Colorado trip sounds amazing - is that one of 
       those \"escape everything and find yourself\" 
       trips or more adventure focused?
       
       Also, are you accepting new hiking buddies? 
       Asking for a friend (it's me) 😊'
       
       ENERGY: Playful, shows interest, suggests 
       future activity
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
       
       REPLY 2 (Genuine + Curious):
       
       'That's awesome. Honestly, your passion for 
       this stuff is genuinely attractive. 
       
       What's the Colorado trip about? And real 
       question - would you ever want to grab coffee 
       before you go? I feel like I'm genuinely 
       curious about you, and it seems like we're 
       already vibing.'
       
       ENERGY: Sincere, direct interest, suggests 
       meeting
       
       ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
       
       REPLY 3 (Curious + Building):
       
       '20 trails and a Colorado trip? I have so many 
       questions. Most importantly: is this trip 
       something you'd want a hiking partner for, or 
       is it more of a solo spiritual journey thing?
       
       (Asking because I might know some good trails 
       there...)'
       
       ENERGY: Playful curiosity, shared interest, 
       suggests future activity"

User A: "I like a mix - the humor from 1 but the 
         directness from 2"

Moly: "Smart combination. Here's a hybrid:
       
       'That's awesome. 20 trails with that energy 
       is genuinely impressive.
       
       Real talk though - I'm finding myself 
       genuinely curious about you. Would you want 
       to grab coffee before the Colorado trip? 
       I feel like we're already vibing 😊'
       
       This keeps the casual tone but shows real 
       interest. Better?"

User A: "Perfect!"

Moly: "✓ Saved as 'Sarah - Reply v1'
       
       This should keep momentum with her. 
       Good luck!"
```

---

### 4.3 Flow 3: Tone Switching Mid-Conversation

```
SCENARIO: User A wants to change approach after 
          receiving feedback or realization

User A: "Wait, I've been messaging Sarah for 2 weeks
         and we haven't met. I think I'm being too 
         casual. Can you help me be more direct about 
         wanting to actually meet?"

Moly: "Absolutely. That's smart - momentum matters.
       
       What would feel most authentic to you?
       
       A) More direct/confident
       B) More sincere/vulnerable
       C) Both (direct + vulnerable)
       
       This is important - what actually feels true 
       to how you feel about her?"

User A: "C - I do like her and I want to be real 
         about it"

Moly: "Got it. Here's a message that shows you 
       care AND want to move things forward:
       
       'Sarah, I realize I've been keeping things 
       pretty casual and I want to be real with you.
       I actually like you. Not just vibing-online 
       like, but interested-in-really-knowing-you 
       like.
       
       Are you open to grabbing coffee or doing 
       something this weekend? I want to know if 
       this energy translates in person.'
       
       BREAKDOWN:
       ✓ Vulnerable (admits feelings)
       ✓ Direct (what you want)
       ✓ Confident (not apologetic)
       ✓ Clear ask (removes ambiguity)
       ✓ Shows intention (serious intent)
       
       Does this feel right?"

User A: "Yes, exactly what I needed!"
```

---

## 5. ADVANCED FEATURES

### 5.1 Message Comparison Tool

```
COMPARE MESSAGES SIDE-BY-SIDE

[Message Comparison View]

┌──────────────────────┬──────────────────────┐
│ VERSION 1 (Playful) │ VERSION 2 (Sincere)  │
├──────────────────────┼──────────────────────┤
│ "Your dog photo is  │ "I know this might   │
│  the best profile   │  sound forward, but  │
│  pic I've seen.     │  I'm not great at    │
│  Tell me about Max?"│  casual. I saw your  │
│                    │  profile and felt    │
│ TONE: Light, fun    │  like I should reach │
│ VIBE: Compliment    │  out."              │
│       focused       │                      │
│ RESPONSE: Easy      │ TONE: Sincere,      │
│           (talk     │        vulnerable   │
│            about    │ VIBE: Direct        │
│            dog)     │       intention     │
│                    │ RESPONSE: Emotional │
│ RISK: Might seem   │           (shares   │
│       surface       │            feelings)│
│ SUCCESS RATE: 70%   │ SUCCESS RATE: 65%   │
└──────────────────────┴──────────────────────┘

USER DECIDES: "I like v1's approachability 
but v2's sincerity. Can we mix them?"

Moly generates hybrid version.
```

### 5.2 A/B Testing Suggestions

```
SCENARIO: User A wants to test what works

User A: "I have 3 versions. Which should I send 
         first? Or should I try different ones with 
         different people?"

Moly: "Good thinking. Here's my recommendation:
       
       STRATEGY 1: Send best version
       └─ Send MESSAGE 1 (highest success probability)
       └─ If she responds, great
       └─ If not, learn from it
       
       STRATEGY 2: Test with multiple matches
       └─ Send MESSAGE 1 to Sarah (serious match)
       └─ Send MESSAGE 2 to Jessica (casual match)
       └─ Send MESSAGE 3 to Emily (friend vibe)
       └─ See what resonates with different people
       
       STRATEGY 3: Serial testing
       └─ Send MESSAGE 1 this week
       └─ Analyze response
       └─ Adjust MESSAGE 2 for next week
       └─ Iterate and improve
       
       Which feels right for you?"

User A: "Strategy 2 - I want to see what works with 
         different types"

Moly: "Smart. Here's your testing plan:

       Sarah (potential serious):
       Send: 'That's awesome. 20 trails...' [Sincere]
       
       Jessica (casual vibe):
       Send: 'Okay so I have to ask...' [Playful]
       
       Emily (friend energy):
       Send: 'What's the coolest...' [Curious]
       
       Track responses and come back - I'll help you 
       analyze what worked!"
```

### 5.3 Message Confidence Score

```
MOLY ANALYZES MESSAGE QUALITY

When generating messages, Moly shows:

MESSAGE 1
├─ Confidence score: 78%
├─ Why high: Witty opener, specific reference, 
│   clear ask, easy to respond to
├─ Strengths: Memorable, funny, shows attention
├─ Weaknesses: Might be too casual for serious 
│            intent
└─ Best for: Casual, friend-first, testing waters

MESSAGE 2
├─ Confidence score: 82%
├─ Why high: Shows genuine interest, vulnerable, 
│   direct, moves toward meeting
├─ Strengths: Sincere, clear intention, builds 
│            connection
├─ Weaknesses: Requires matching energy to land
└─ Best for: Serious interest, potential LTR

MESSAGE 3
├─ Confidence score: 71%
├─ Why: Playful but less personal connection
├─ Strengths: Low pressure, fun, safe
├─ Weaknesses: Might get lost in other messages
└─ Best for: Second/third message, momentum

USER INSIGHT: Pick message 2 for serious intent
```

---

## 6. SAFETY & LEGAL SAFEGUARDS

### 6.1 What Moly Chat NEVER Does

```
❌ NEVER generates manipulative messages
   - No "playing hard to get" tactics
   - No manufactured jealousy
   - No negging or insults
   - No deceptive flattery
   
❌ NEVER analyzes User B's psychology
   - Can't generate "this will make them chase you"
   - Can't create psychological manipulation
   - Can't suggest exploiting their vulnerabilities
   
❌ NEVER suggests dishonesty
   - Can't help fabricate common interests
   - Can't suggest lying about intentions
   - Can't create false personas
   
❌ NEVER generates harassment
   - Can't persist after someone says no
   - Can't send repeated messages to uninterested 
     person
   - Can't craft aggressive/coercive language
   
❌ NEVER stores User B data in profiles
   - Only stores what User A tells us they know
   - No behavioral analysis of User B
   - No tracking of User B's replies
   - No patterns about User B's psychology
```

### 6.2 Ethical Guidelines Built In

```
MOLY'S ETHICAL FRAMEWORK

When generating messages, Moly ensures:

✅ AUTHENTICITY
   "Does this feel true to who User A is?"
   Refuses to generate fake personas

✅ RESPECT
   "Is this respectful to User B?"
   Refuses to suggest games or manipulation

✅ CONSENT-FOCUSED
   "Does this respect User B's autonomy?"
   Messages should enable clear choice, not coerce

✅ HONEST INTENT
   "Is User A being honest about what they want?"
   Refuses to hide true intentions

✅ SAFETY
   "Could this be used to harm?"
   Refuses dangerous or predatory language

Example: If User A says:
"I want to manipulate Sarah into liking me"

Moly responds:
"I can't help with manipulation. But I can help 
you show your genuine self in a confident way - 
which is actually way more effective.

Want to try that instead?"
```

---

## 7. TECHNICAL ARCHITECTURE

### 7.1 Chat Backend (What Happens Behind Scenes)

```
USER MESSAGE FLOW:

1. User A types message in chat
   ↓
2. Sent to Claude API with CONTEXT:
   {
     "user_intention": "dating_serious",
     "user_b_profile": {
       "name": "Sarah",
       "interests": ["hiking", "dogs", "coffee"],
       "user_a_observations": "adventurous, genuine"
     },
     "user_a_style": {
       "tone": "witty",
       "secondary": "sincere",
       "preference": "authentic"
     },
     "conversation_history": [previous exchanges],
     "stage": "initial_opening"
   }
   ↓
3. Claude generates suggested messages based on:
   - User A's stated preferences
   - User B's profile (as described by User A)
   - Conversation context
   - Ethical guidelines
   ↓
4. Moly ranks messages by:
   - Authenticity to User A
   - Likelihood of positive response
   - Clarity of intention
   - Ease of responding
   ↓
5. Presents to User A with analysis and coaching
```

### 7.2 Storage (What Gets Saved)

```
STORED LOCALLY ON USER A'S DEVICE:

✓ Chat conversation history
  "What was User A's stated intention with Sarah?"
  "What tone did they prefer?"
  "What messages did we generate?"

✓ Message versions & edits
  "Message v1 for Sarah (sent)"
  "Message v2 for Sarah (not sent)"
  "Message v3 for Sarah (draft)"

✓ User A's preferences (learned over time)
  "User A prefers witty + sincere (70% use)"
  "User A typically uses messaging for dating (not hookups)"
  "User A values authenticity"

✓ Contact notes updated
  {
    "sarah": {
      "first_contacted": "Aug 4 via Moly chat",
      "opening_message_tone": "witty + sincere",
      "response": "Positive",
      "ongoing": true
    }
  }

NOT STORED ANYWHERE:

❌ User B's actual messages (they read them in dating app)
❌ User B's behavioral analysis
❌ Moly's predictions about User B
❌ Analysis of User B's personality
```

---

## 8. USER WORKFLOWS

### 8.1 Workflow A: First-Time User (Cold Opening)

```
┌─ User opens Moly Chat
│
├─ "Who do you want to message?" → Sarah
│
├─ "Tell me about Sarah" → Inputs observations
│
├─ "What's your intention?" → Dating/serious
│
├─ "How do you communicate?" → Witty + sincere
│
├─ Moly generates 3 messages
│
├─ User picks one, or asks for variations
│
├─ "Ready to send?" → Yes
│
├─ Message saved to Sarah's contact
│
└─ Conversation archived for future reference
```

### 8.2 Workflow B: Following Up Reply

```
┌─ Sarah replies to opening message
│
├─ Moly chat detects reply (optional)
│
├─ "Sarah replied! Help respond?" → Yes
│
├─ Moly loads:
│  - Sarah's profile (from first message)
│  - User A's previous message
│  - Conversation history
│  - User A's style preferences
│
├─ Moly analyzes Sarah's reply:
│  "She seems engaged, asking questions"
│
├─ Generates context-aware replies
│
├─ User picks and sends
│
└─ Conversation continues
```

### 8.3 Workflow C: Pivoting Tone

```
┌─ User messaging someone casually
│
├─ User A decides they want to be more serious
│
├─ "I want to pivot to more sincere tone" 
│  (in Moly chat)
│
├─ Moly generates new messages with:
│  - More vulnerability
│  - More direct intention
│  - More genuine interest
│
├─ User picks new tone
│
├─ "Here's how to transition smoothly..."
│
└─ Moly suggests transition message
```

---

## 9. UI/UX SPECIFICATIONS

### 9.1 Chat Interface Details

```
MESSAGE INPUT OPTIONS:

1. Free text typing
   └─ Type full message
   └─ Moly interprets and generates replies

2. Quick suggestions
   └─ "Witty + sincere"
   └─ "Direct & honest"
   └─ "Curious & engaging"
   └─ Moly generates based on preset

3. Guided questions (for detailed requests)
   ├─ "What's their vibe?"
   ├─ "What's your intention?"
   ├─ "How do you want to come across?"
   └─ Moly generates from answers

4. Voice input (future)
   └─ "Tell me about Sarah" (voice)
   └─ Moly transcribes and understands
```

### 9.2 Output Format

```
WHEN MOLY GENERATES MESSAGES:

┌─────────────────────────────────────┐
│ MESSAGE N (Tone: Witty + Sincere)  │
├─────────────────────────────────────┤
│                                     │
│ [Actual message text]               │
│                                     │
│ CONFIDENCE: 82%                     │
│                                     │
│ WHY THIS WORKS:                     │
│ • Makes them laugh (opener)         │
│ • Shows you paid attention          │
│ • Clear interest                    │
│ • Easy to respond to                │
│                                     │
│ BEST FOR:                           │
│ • Serious interest matches          │
│ • People who value humor + depth    │
│                                     │
│ [Copy] [Save] [Adjust]              │
│                                     │
└─────────────────────────────────────┘
```

### 9.3 Saved Messages Library

```
USER SEES:

My Message Versions

Sarah (Tinder):
├─ Opening v1 (sent) - "Witty + sincere"
│  Response: ✓ Positive, she asked questions
├─ Reply v1 (sent) - "Curious + engaged"
│  Response: ✓ Positive, continuing conversation
└─ Reply v2 (saved draft) - "More direct"
   Status: Ready if needed

Jessica (Bumble):
├─ Opening v1 (sent) - "Playful"
│  Response: ✗ No response (still waiting)
├─ Opening v2 (option) - "Sincere"
│  Status: Alternative ready
└─ Opening v3 (discarded) - "Too casual"

Emily (Hinge):
└─ Opening v1 (draft) - "Question-based"
   Status: Not sent yet

[View all] [Archive old] [Compare versions]
```

---

## 10. MESSAGING TEMPLATES & EXAMPLES

### 10.1 Opening Messages by Context

```
FORMAL CONTEXT (Professional/Business):
──────────────────────────────────────
"Hi Sarah, I've been following your work in 
marketing and I'm impressed by your approach to 
community engagement. I'm currently exploring 
opportunities in that space and would value your 
insights. Would you be open to a brief call?"

FRIENDLY CONTEXT (Social/Friend-Making):
────────────────────────────────────────
"Hey Sarah! I noticed you love hiking - I've been 
looking for people to explore trails with. Are you 
usually solo or do you hike with groups? Would be 
fun to connect."

DATING CONTEXT (Romantic Interest):
───────────────────────────────────
"Your profile genuinely caught my attention. You 
seem like someone who actually lives life fully, 
and that's rare. I'd love to grab coffee and get 
to know you."
```

### 10.2 Follow-up Replies by Context

```
FORMAL CONTEXT REPLY:

THEM: "That sounds interesting. What's your background?"
YOUR REPLY:
──────────
"I have [X years] experience in [field]. I'm 
particularly focused on [specific area]. I'd love 
to hear how you approach this. Are you free for a 
30-min call next week?"

---

FRIENDLY CONTEXT REPLY:

THEM: "Yeah, I love hiking! What area do you usually go?"
YOUR REPLY:
──────────
"Mostly around [area], but I'm always looking for 
new spots. Do you have any favorites? Also, are 
you planning any group hikes soon?"

---

DATING CONTEXT REPLY:

THEM: "That sounds fun! What do you like to do?"
YOUR REPLY:
──────────
"A lot! I love hiking, trying new coffee places, 
and honestly just having real conversations. What 
about you? Would love to know what drives you."
```

---

## 11. MONETIZATION

### 11.1 Free vs. Pro Tiers

```
FREE TIER:
├─ 10 message generations per day
├─ 3 messages per generation (choose from)
├─ Basic tone options (3-5)
├─ No saved message history
└─ Limited to first 2 conversations per contact

PRO TIER ($4.99/month):
├─ Unlimited message generations
├─ 5+ message options per generation
├─ All tone combinations
├─ Full message history (saved forever)
├─ Unlimited contacts
├─ Conversation coaching (when they reply)
├─ Message A/B testing suggestions
├─ Tone comparison tool
└─ Priority support

PREMIUM TIER ($9.99/month):
├─ Everything in Pro, plus:
├─ Advanced conversation analytics
│  (what tone works best with them)
├─ Voice chat with Moly (talk out messages)
├─ Real-time suggestion improvements
├─ Personalized dating coaching
├─ Message performance tracking
├─ Cross-platform message sync
└─ Quarterly coaching sessions
```

---

## 12. IMPLEMENTATION ROADMAP

### Phase 1: MVP Chat (Weeks 1-6)
- [ ] Basic chat interface
- [ ] Claude API integration
- [ ] Generate 3 message versions
- [ ] 5 core tones
- [ ] Save messages locally
- [ ] Mobile chat support

### Phase 2: Enhanced Features (Weeks 7-12)
- [ ] Tone comparison tool
- [ ] Conversation history
- [ ] A/B testing suggestions
- [ ] Confidence scores
- [ ] More tone combinations
- [ ] Voice input (future)

### Phase 3: Advanced (Weeks 13-18)
- [ ] Real-time suggestions on reply
- [ ] Cross-platform sync
- [ ] Analytics dashboard
- [ ] Personalization learning
- [ ] Voice coaching (future)

---

## 13. SAFETY & COMPLIANCE

### Ethical AI Use
- ✓ Messages are authentic to User A
- ✓ No manipulation tactics
- ✓ Respects User B autonomy
- ✓ Honest about intentions
- ✓ Refuses deceptive content

### User Data
- ✓ Chat history stays local
- ✓ No User B data stored long-term
- ✓ No behavioral profiling
- ✓ Users can delete anytime
- ✓ Full transparency on what's stored

---

## SUMMARY

Moly's natural language chat transforms message generation from passive suggestions to active coaching across ANY communication context. Users have a conversational partner who:

1. **Understands purpose** - Is this Formal, Friendly, or Dating communication?
2. **Gathers context** - Who is the recipient and what's your relationship goal?
3. **Generates options** - Multiple message versions for each context
4. **Explains choices** - Why this approach works best
5. **Learns preferences** - Gets better understanding of user's style over time
6. **Maintains authenticity** - Always genuine, never manipulative
7. **Switches contexts** - User can go from dating to professional to friend messages

This makes Moly not just a dating tool, but a universal communication coach that helps users craft genuine, authentic messages across professional, social, and romantic contexts.

---

**Version:** 1.0  
**Status:** Ready for Design & Implementation  
**Last Updated:** August 4, 2026
