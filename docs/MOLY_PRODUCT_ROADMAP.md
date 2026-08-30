# MOLY: Product Roadmap & Safe Operating Guidelines
**Last Updated:** August 2026  
**Status:** Ready for Development

---

## 1. PRODUCT OVERVIEW

### Vision
Moly is a dating confidence coaching app that helps users write better messages. It automates the copy-paste workflow of consulting an AI for dating advice, making it seamless and integrated into the dating experience.

### Core Value Proposition
"Write better dating messages. Get better responses. Feel more confident."

### Target Users
- People using dating apps (Tinder, Bumble, Hinge, Feeld, etc.)
- Ages 18-45, primarily mobile users
- People who experience dating anxiety or want to improve messaging quality
- Both men and women across all sexual orientations

### Success Metrics
- User retention (DAU/MAU)
- Messages coached per user per day
- Conversion to paid tier
- User feedback on confidence improvement

---

## 2. CORE FUNCTIONALITY

### 2.1 Primary Feature: Reply Coaching

**How It Works:**
1. User A receives message from User B in dating app
2. User A copies the incoming message
3. User A opens Moly app
4. Moly detects clipboard (optional, nice-to-have)
5. User A pastes message or it auto-fills
6. Moly generates 2-3 read-only suggestions in different tones
7. User A reads suggestions (for inspiration/coaching)
8. User A opens reply field in dating app
9. User A types their own response (influenced by suggestions)
10. User A sends message through dating app

**Key Design Principle:** User A always types the actual message sent. Moly never sends anything.

### 2.2 Secondary Feature: Opening Message Coaching

**How It Works:**
1. User A decides to message someone new
2. User A tells Moly what they know about User B (manually entered)
   - Example: "She likes hiking and marketing, seems outdoorsy"
3. Moly coaches on opening strategies
4. Moly shows approach examples (for inspiration only, not to copy)
   - "Consider: common interest opener"
   - "Consider: engaging question"
   - "Consider: personal story"
5. User A types their own opening message
6. Moly reads what User A wrote, suggests improvements
7. User A sends through dating app

**Key Design Principle:** User A manually provides context. Moly coaches approach, not content. User A always writes.

### 2.3 Tertiary Feature: Conversation Dashboard (Future)

**What It Tracks (About User A's Participation):**
- How often User A messages (engagement frequency)
- How balanced the conversation is (message send/receive ratio)
- Topic diversity (what topics are discussed)
- Conversation momentum (escalating, stable, declining)
- Response patterns (when User A typically replies)

**What It Shows User A:**
- "You're engaging well in this conversation"
- "Balance is healthy (you're not doing all the talking)"
- "Momentum is positive - conversation is deepening"
- "Consider asking a question - you haven't in your last 3 messages"
- "This person engages most with [topic you both enjoy]"

**Key Design Principle:** All feedback is about User A's participation and coaching, NOT about scoring/profiling User B.

---

## 3. WHAT MOLY CAN DO ✅

### Data Moly Can Store (User A Only)
- ✅ User A's app preferences (tone preferences, suggestion style, language)
- ✅ User A's manual notes about people they're chatting with
  - User A manually enters: "Sarah - likes hiking, works in marketing"
- ✅ User A's account data (email, password, account creation date)
- ✅ User A's subscription status (free vs. pro tier)
- ✅ Behavioral metrics about User A's app usage
  - "User A requested X suggestions today"
  - "User A uses empathy suggestions 70% of the time"
  - "User A spends average 45 seconds reading suggestions"
- ✅ Aggregate analytics (anonymized, for product improvement)
- ✅ Engagement tracking for User A (DAU, retention, feature usage)

### Content Moly Can Generate
- ✅ Suggestion responses in different tones for incoming messages
  - Casual/friendly
  - Formal/professional
  - Witty/playful
  - Direct/confident
  - Vulnerable/authentic
- ✅ Coaching suggestions for opening strategies
  - Common interest approach
  - Question-based approach
  - Personal story approach
  - Direct/confident approach
- ✅ Coaching feedback on messages User A wrote
  - "Good, you asked a question"
  - "Consider adding something personal"
  - "This tone matches their energy"
- ✅ Educational content about dating communication
  - Dating safety tips
  - Common conversation patterns
  - Tips for better first messages

### Features Moly Can Provide
- ✅ Clipboard monitoring (notify user when message copied)
- ✅ Suggestion toggling (show/hide different tone options)
- ✅ Conversation dashboard (User A's engagement metrics)
- ✅ Custom note-taking (User A's notes about people)
- ✅ Tone preference selection (User A chooses preferred suggestion style)
- ✅ Analytics for User A (how they're progressing)
- ✅ Fraud detection on User A (detect if User A is scamming others)

---

## 4. WHAT MOLY CANNOT DO ❌

### Data Moly Must NEVER Store
- ❌ Incoming messages from User B (message content)
- ❌ Outgoing messages sent by User A (message content)
- ❌ Conversation history between User A and User B
- ❌ User B's profile data (from dating app)
- ❌ Detailed chat logs or transcripts
- ❌ Recording of suggestion generation

### Content Moly Must NEVER Generate
- ❌ Auto-send messages (User A must type and send)
- ❌ Complete messages that User A just copies (defeats coaching purpose)
- ❌ Messages impersonating User A's authentic voice
- ❌ Manipulative tactics or pickup artist language
- ❌ Messages designed to exploit User B's known preferences

### Features Moly Must NEVER Build
- ❌ Profiling/scoring of User B
  - NOT: "User B compatibility: 73%"
  - NOT: "User B engagement score: 8/10"
- ❌ Psychological analysis of User B
  - NOT: "User B seems to have anxious attachment"
  - NOT: "User B is likely insecure"
- ❌ Automated red flag detection about specific users
  - NOT: "⚠️ User B might be a scammer"
- ❌ Behavioral tracking of User B's messaging patterns
- ❌ Automated blocking/reporting of other users (without human review)
- ❌ Machine learning models trained on conversation content
- ❌ Integration with other apps/platforms to share user data
- ❌ Advertising or selling user data

---

## 5. LEGAL FRAMEWORK

### 5.1 Core Legal Principle
**Moly is a coaching tool for User A, not a content provider or platform moderator.**

User A is solely responsible for:
- All messages they send
- Content and tone of their communication
- Decisions about who to message and how
- Safety and outcomes of their dating interactions

Moly is not responsible for:
- Content of User A's messages
- How User B responds
- Relationship outcomes
- Misuse of coaching suggestions

### 5.2 Why This Model is Legal

**Copy-Paste Equivalence:**
Just as it's legal for User A to:
1. Copy message from dating app
2. Paste into ChatGPT
3. Get suggestions
4. Close ChatGPT
5. Type inspired response in dating app
6. Send it

It's equally legal for User A to:
1. Copy message from dating app
2. Paste into Moly
3. Get suggestions
4. Type inspired response in dating app
5. Send it

The legal exposure is identical because User A types and sends everything.

**Key Legal Points:**
- ✅ User A owns their own communications
- ✅ User A can consult any AI for advice
- ✅ Consultation is private (no disclosure to User B required)
- ✅ User A is responsible for their message content
- ✅ User A made conscious, intentional choices
- ✅ No data is stored about the interaction
- ✅ No third party (User B) was profiled

### 5.3 User Consent Framework

**Required for Launch:**
- [ ] Terms of Service (clear liability disclaimers)
- [ ] Privacy Policy (what data is/isn't stored)
- [ ] First-run user education (explain coaching model)
- [ ] Clear UI language ("Coaching suggestions. You write the message.")

**First-Run Consent Flow:**
```
Screen 1: "About Moly"
"Moly helps you write better dating messages. You always 
write and send your own messages. Moly provides suggestions 
and coaching."

Screen 2: "You Are In Control"
"☑ I understand Moly is a coaching tool, not a ghostwriter
☑ I write all my own messages
☑ I am responsible for what I send
☑ I have read the Terms of Service"

[Only proceed if all checked]
```

### 5.4 What User B Does NOT Need to Know

User B does not need to be informed that:
- User A is using a coaching app
- User A consulted an AI before responding
- Moly helped User A think about tone

**Why:**
- This is equivalent to User A asking a friend for advice before responding
- User A's message is authentically written by User A
- The process of how User A decided what to write is User A's business
- No one discloses their advice sources in dating

**Optional:** User A can disclose if they choose ("I use an app to help me be my best self"), but it's not required or necessary.

### 5.5 Platform Terms of Service Compliance

**Dating Platform ToS Risks:**
Most dating apps prohibit:
- Automated tools accessing messages
- Bots or third-party integrations
- Non-human content generation

**Moly Compliance:**
- ✅ No automated access (user manually copies/pastes)
- ✅ No integration with platform APIs (standalone app)
- ✅ No message generation (user types everything)
- ✅ No bot behavior (user actively using app)
- ✅ Read-only suggestions (user controls all output)

**Risk Mitigation:**
- Launch with browser extension first (less friction from platforms)
- Mobile app as secondary (copy-paste workflow is safe)
- Don't violate platform ToS (no scraping, no bots, no APIs without permission)

### 5.6 Fraud Detection Framework

**What Moly CAN Do:**
- ✅ Detect User A showing fraud patterns (scamming others)
  - Example: Repeatedly requesting money suggestions across multiple users
- ✅ Block User A for violating ToS
- ✅ Report to law enforcement (if serious fraud detected)

**What Moly CANNOT Do:**
- ❌ Flag User B as a scammer
- ❌ Warn User A about specific users
- ❌ Create "scam scores" for users

**Fraud Detection Algorithm:**
- Track User A's suggestion requests for financial/urgent language
- Detect pattern repetition across many users in short timeframe
- Require human review before blocking
- Transparency: warn User A before final action
- Law enforcement report (only if Fraud Score > 85)

---

## 6. TECHNICAL ARCHITECTURE

### 6.1 Tech Stack

**Frontend:**
- React Native (iOS + Android cross-platform)
- Flutter (alternative option)
- Browser extension (Chrome/Firefox/Safari for web version)

**Backend:**
- Claude API (suggestions generation)
- Basic auth & user management
- Encrypted local storage (device-first design)
- Optional: Simple database for preferences only

**Infrastructure:**
- API-first design (minimal server load)
- No message storage (process and discard immediately)
- Encrypted data in transit
- User data encryption at rest

### 6.2 Data Flow

```
USER A INPUTS:
↓
Clipboard paste or typed message
↓
MOLY PROCESSING:
↓
Send to Claude API
(Prompt: "Generate 2-3 response suggestions in different tones")
↓
Receive suggestions
↓
Display in app (read-only)
↓
USER A DECISION:
↓
Reads suggestion (or ignores)
↓
Types own message in dating app
↓
Sends via dating app
↓
MOLY STORAGE:
↓
Delete suggestion from memory (after 5-10 minutes)
Store: User A's preferences/analytics only
Do NOT store: Message content, User B data, conversation
```

### 6.3 Storage Architecture

**Local Device Storage (Prioritized):**
- User A's preferences (tone, style, language)
- User A's manual notes about people
- Timestamp of last use
- Downloaded analytics

**Server Storage (Minimal):**
- User account (email, password hash, subscription)
- User A's preferences (synced across devices)
- Aggregate analytics (anonymized)
- Fraud detection flags for User A

**NEVER Store:**
- Message content
- User B's data
- Conversation history
- Generated suggestions (process and discard)

### 6.4 Privacy-First Design

**Zero Conversation History:**
- Suggestions generated in Claude API
- Received by app
- Displayed to user
- Deleted from memory after 10 minutes
- Never persisted

**No Message Logging:**
- Don't record what User A typed
- Don't record what User B said
- Don't track who messaged whom
- Only track: "User A requested coaching" (aggregate metric)

**User Data Rights:**
- User can delete their account (all data deleted)
- User can delete notes about specific people
- User can request data export (limited, since minimal stored)
- User can opt-out of analytics

---

## 7. MONETIZATION STRATEGY

### 7.1 Freemium Model

**Free Tier:**
- 10 suggestions per day
- 2 basic tone options (casual, formal)
- Opening message coaching (basic)
- No analytics
- No conversation dashboard

**Pro Tier ($4.99/month):**
- Unlimited suggestions
- 6 tone options (casual, formal, witty, direct, vulnerable, confident)
- Advanced opening strategies
- Conversation dashboard (User A's engagement)
- Conversation history (User A's past chats with person - optional)
- Priority support
- Remove ads (if any)

**Premium Tier ($9.99/month) - Future:**
- Everything in Pro
- Personal analytics dashboard
- AI-powered insights on User A's communication style
- Personalized coaching (based on User A's preferences)
- Advanced tone customization
- Integration with calendar (suggest when to message)

### 7.2 Revenue Opportunities (Later)

**Analytics for Insights:**
- Sell anonymized, aggregate insights to productivity apps
- Example: "Dating app users get 23% more responses with AI-assisted coaching"
- Comply with privacy laws (no individual-level data)

**B2B Partnerships:**
- Partner with dating apps (revenue share model)
- Licensing to dating platforms
- White-label version for dating apps

**Premium Features:**
- Personal dating coach (human review of suggestions)
- Integration with other lifestyle apps
- Advanced analytics for serious relationships

### 7.3 Pricing Psychology

- **$4.99/month** feels like a "toy" purchase (low friction)
- **$59.88/year** feels like better value (annual option)
- **Freemium conversion:** Target 5-10% of free users → paid
- **Lifetime value target:** $50-200 per paid user

---

## 8. PRODUCT WORKFLOWS

### 8.1 Workflow 1: Reply Message Coaching

**Scenario:** User A receives message, wants to respond

```
1. USER A IN DATING APP
   Receives message: "How was your weekend?"
   ↓
2. USER A OPENS MOLY
   Copies message OR Moly detects clipboard
   ↓
3. MOLY PROCESSES
   Sends to Claude API with prompt:
   "Generate 2-3 response suggestions for this message.
   Tones: casual, formal, witty"
   ↓
4. MOLY DISPLAYS
   Shows 3 suggestions (read-only, non-selectable)
   - "It was great! Went hiking, tried a new restaurant. You?"
   - "Pretty good! How about yours?"
   - "Not bad! What's new with you?"
   ↓
5. USER A READS
   Reads suggestions (15-20 seconds)
   ↓
6. USER A CLOSES MOLY
   ↓
7. USER A IN DATING APP
   Opens reply field
   ↓
8. USER A TYPES
   Types own response, influenced by suggestions
   Could be exact match, modified, or completely original
   ↓
9. USER A SENDS
   Sends message through dating app
   ↓
10. MOLY CLEANUP
   Delete suggestions from memory
   Store only: "User A requested coaching at [time]"
```

**Key Points:**
- User A types everything
- Moly never sends anything
- Suggestions are read-only
- No data stored about User B
- User A owns the response

### 8.2 Workflow 2: Opening Message Coaching

**Scenario:** User A wants to message someone new

```
1. USER A OPENS MOLY
   Clicks "Compose Opening"
   ↓
2. MOLY ASKS
   "What do you know about this person?"
   ↓
3. USER A ENTERS (Manually)
   "Sarah - 28 - likes hiking and coffee - marketing executive"
   ↓
4. MOLY COACHES
   Shows approach examples:
   - "Find common interest: 'I saw you love hiking!'"
   - "Ask engaging question: 'What's your go-to weekend?'"
   - "Personal approach: 'I'm really into hiking too...'"
   
   Text: "These are approaches, not messages to copy. 
          Write something that's authentically you."
   ↓
5. USER A WRITES
   Types their own opening message
   "I love that you're into hiking! Have you explored [trail]?"
   ↓
6. MOLY SUGGESTS IMPROVEMENTS
   Reads what User A wrote
   "Great! You mentioned common interest and asked a question.
    Consider adding something personal about yourself?"
   ↓
7. USER A REFINES
   Edits message if wanted
   ↓
8. USER A SENDS
   Sends through dating app
   ↓
9. MOLY CLEANUP
   Store: User A's note about Sarah
   Do NOT store: The message User A sent
```

**Key Points:**
- User A manually provides context about User B
- Moly suggests approaches, not message text
- User A types everything
- Feedback is on User A's writing, not User B
- User A owns the opening message

### 8.3 Workflow 3: Conversation Dashboard (Premium)

**Scenario:** User A wants to see how they're doing in a conversation

```
1. USER A OPENS MOLY
   Goes to Conversation Dashboard
   ↓
2. MOLY DISPLAYS (About User A's Participation)
   
   "Conversation with Sarah"
   
   📈 Engagement: Balanced
   (You send 1.2 messages per their message - good ratio)
   
   💬 Topics: Outdoors, Work, Lifestyle
   (You're matching their interests)
   
   ⏱️ Timing: Evening responses
   (You respond around 8-10pm, they respond 7-11pm)
   
   🎯 Momentum: Positive & Deepening
   (Messages going from casual to more personal)
   
   ⚠️ Suggestions:
   - "You haven't asked them a question in 2 messages"
   - "Consider sharing something personal"
   - "They engaged most when you talked about [topic]"
   
3. USER A DECIDES
   Can see their own patterns
   Can use insights for next message
   
4. MOLY DOES NOT:
   ✗ Score Sarah
   ✗ Analyze Sarah's behavior
   ✗ Create profile about Sarah
   ✗ Give advice about Sarah's character
```

**Key Points:**
- All metrics are about User A's participation
- No profiling of User B
- Coaching for User A's improvement
- Insights help User A communicate better

---

## 9. USER INTERFACE PRINCIPLES

### 9.1 Clarity Over Feature Richness

**UI Must Clearly Show:**
- "These are suggestions, not messages to copy"
- "You always write your own message"
- "Moly is coaching, not ghostwriting"
- "You control all decisions"

**Avoid:**
- ❌ Copy buttons (no one-click copying)
- ❌ Send buttons in Moly (can't send from app)
- ❌ Autofill that looks like ghostwriting
- ❌ Suggestion text that looks like User A's voice

### 9.2 Suggestions Display

```
[Message from User B displayed]

💡 COACHING SUGGESTIONS (Read-only)
ⓘ Tap any suggestion to think about it. You'll type your own message.

1️⃣  Casual/Friendly Tone:
    "It was amazing! Went hiking 
     and tried this new spot. You?"
    [← Not copyable, just for reading]

2️⃣  Direct/Confident Tone:
    "Great weekend. Yours?"
    [← Not copyable, just for reading]

3️⃣  Witty Tone:
    "Too good to explain. Let's 
     grab coffee and I'll tell you?"
    [← Not copyable, just for reading]

ⓘ Now go type your response in [Dating App]
```

### 9.3 Opening Message Coaching Display

```
COMPOSE OPENING MESSAGE

"What do you know about this person?"
[Text field: "Sarah - hiking - marketing - outdoorsy"]

SUGGESTED APPROACHES:
(Not messages to copy - just inspiration)

✓ Common Interest Approach
  Example: "I love hiking! Have you 
           explored [local trail]?"
  
✓ Question Approach
  Example: "What's your ideal weekend?"
  
✓ Personal Story Approach
  Example: "I'm really into the outdoors. 
           Would love to hear about your 
           favorite hikes."

Your Opening:
[Text field for User A to type]

💬 Read to send? Review it below.

[When User A types message]

QUICK FEEDBACK:
✓ You mentioned a common interest (good!)
✓ You asked a question (great engagement!)
? Consider adding something personal about yourself?
```

### 9.4 Dashboard Display

```
CONVERSATION INSIGHTS (Your Participation)

Sarah ••• Started 3 days ago

📊 ENGAGEMENT
   You ╥ 47 messages
  They ╥ 38 messages
  Balance: Healthy (you're not dominating)

💬 TOPICS
   Most engaged: Outdoors, Work
   Least engaged: Politics, Family
   
⏱️ PATTERNS
   You respond: ~45 minutes average
   They respond: ~1 hour average
   Best time to message: 8-9pm

📈 MOMENTUM
   ↗ Positive (moving toward personal topics)
   
🎯 SUGGESTIONS FOR YOU
   ✓ Keep sharing personal stories - they respond well
   ✓ You're asking good questions - keep going
   ? Try asking about their values or dreams next
   ✓ Your tone is matching theirs well

⚠️ COACHING
   "You've been doing all the questions lately. 
    Share more about yourself next time!"
```

---

## 10. DEVELOPMENT ROADMAP

### Phase 1: MVP (Weeks 1-8)

**What to Build:**
- [ ] Core app structure (TypeScript + React for browser extension)
- [ ] Authentication (email/password)
- [ ] **Dual-panel UI architecture:**
  - [ ] Chat section (left/top) - conversation with Moly
  - [ ] Suggestions section (right/bottom) - message output
- [ ] **Two dialogue modes:**
  - [ ] Direct mode (fast, immediate suggestions)
  - [ ] Socratic mode (guided questions using Socratic dialogue)
- [ ] Automatic message detection (MutationObserver)
- [ ] Claude API integration (+ local LLM support ready)
- [ ] Suggestion generation (3 versions, different tones)
- [ ] Encrypted local storage (IndexedDB + TweetNaCl.js)
- [ ] Freemium tier system
- [ ] Basic analytics

**Deliverables:**
- Working browser extension (Chrome/Firefox/Safari)
- Automatic message detection on any platform
- Dual-panel UI with chat + suggestions
- Socratic dialogue mode (powered by Socrates library logic)
- Suggestion generation for replies
- Encrypted contact notes (User B profiles)
- Freemium paywall
- Basic legal docs (ToS, privacy)

**Technical Stack:**
- TypeScript + React 18
- Zustand (state)
- Tailwind CSS (styling)
- TweetNaCl.js (encryption)
- IndexedDB (local storage)
- Vite (build)
- Manifest V3 (browser extension)

**Target Launch:** Week 8-9

---

### Phase 2: Browser Extension (Weeks 9-15)

**What to Build:**
- [ ] Chrome/Firefox/Safari extensions
- [ ] Web version (for people using dating sites via browser)
- [ ] Display suggestions inline with dating app
- [ ] Cloud sync (web + mobile)
- [ ] Share sheet integration (iOS)
- [ ] Intent handling (Android)

**Deliverables:**
- Browser extension for web dating
- Better UX than copy-paste (reads visible content)
- Cross-platform feature sync
- Desktop user experience

**Target Launch:** Week 15-16

---

### Phase 3: Opening Messages (Weeks 17-22)

**What to Build:**
- [ ] Opening message coaching workflow
- [ ] Manual profile builder (User A notes about people)
- [ ] Approach suggestions (not message generation)
- [ ] Coaching feedback on User A's openings
- [ ] Storage for User A's notes about people

**Deliverables:**
- Opening message feature working
- Profile note-taking system
- Coaching feedback system

**Target Launch:** Week 22-23

---

### Phase 4: Conversation Dashboard (Weeks 23-32)

**What to Build:**
- [ ] Engagement metrics for User A
- [ ] Conversation state analysis
- [ ] Dashboard UI
- [ ] User A's participation analytics
- [ ] Coaching insights
- [ ] Pro tier features

**Deliverables:**
- Premium dashboard working
- Engagement analytics
- Pro tier monetization live

**Target Launch:** Week 32+

---

### Phase 5: Fraud Detection (Weeks 25-34)

**What to Build (Parallel to Phase 4):**
- [ ] User A behavior tracking (suggestion requests)
- [ ] Fraud pattern detection algorithm
- [ ] Human review workflow
- [ ] User warning system
- [ ] Account blocking/ban system
- [ ] Law enforcement reporting (IC3)

**Deliverables:**
- Fraud detection system live
- Can identify and block scammers
- Reporting to authorities

**Target Launch:** Week 34+

---

### Phase 6: Marketing & Growth (Week 8+ ongoing)

**From Launch:**
- [ ] Product Hunt launch (Phase 1 MVP)
- [ ] Reddit dating communities
- [ ] Twitter/TikTok dating content
- [ ] Influencer partnerships
- [ ] User referral program

**Targets:**
- Week 8-16: 1K-5K users
- Week 16-24: 5K-20K users
- Week 24-32: 20K-50K users
- Week 32+: Growth phase (50K+ users)

---

## 11. LEGAL DOCUMENTS NEEDED

### 11.1 Terms of Service

**Must Include:**
- User A is responsible for all messages
- Moly is a coaching tool, not content provider
- No liability for message content or outcomes
- No data storage of conversations
- User agrees to use app only for legal purposes
- User agrees to not use for fraud/harm
- Platform ToS compliance required
- Account can be banned for violating rules

**Template Structure:**
1. Coaching model (clear explanation)
2. User responsibility
3. Limitation of liability
4. Prohibited conduct
5. Account termination
6. Data privacy
7. Changes to terms

**Estimated:** 1,500-2,000 words

### 11.2 Privacy Policy

**Must Include:**
- What data is collected (account, preferences, analytics only)
- What data is NOT stored (conversations, messages)
- How long data is kept
- User rights (access, deletion, export)
- GDPR/CCPA compliance
- Security measures
- Third-party services (Claude API)
- Contact for privacy questions

**Template Structure:**
1. Data we collect
2. How we use it
3. Data we don't collect
4. Data retention
5. Your rights
6. Security
7. Contact

**Estimated:** 1,500-2,000 words

### 11.3 First-Run Consent

**User Must Accept:**
- [ ] Moly is a coaching tool
- [ ] I write all my own messages
- [ ] I am responsible for what I send
- [ ] Moly doesn't send anything
- [ ] I have read ToS and privacy policy

**No acceptance = can't use app**

---

## 12. MARKETING & POSITIONING

### 12.1 Brand Positioning

**Target Customer:**
"People who feel anxious about dating messaging and want to write with more confidence"

**Key Message:**
"Better messages. Better responses. More confidence."

**Not Positioning As:**
- ❌ "AI writes your messages"
- ❌ "Guaranteed to get dates"
- ❌ "Ghostwriting service"
- ❌ "Manipulation tactics"

**Positioning As:**
- ✅ "Coaching for better communication"
- ✅ "Build dating confidence"
- ✅ "Learn to write better messages"
- ✅ "Get authentic responses"

### 12.2 Value Propositions

**For Anxious Daters:**
"Stop second-guessing yourself. Get suggestions in seconds, write with confidence."

**For People Bad at Texting:**
"Learn messaging patterns that work. Improve your response game."

**For People Overwhelmed by Choices:**
"See different approaches fast. Focus on what feels authentic to you."

### 12.3 Launch Messaging

**Product Hunt:**
"Moly coaches you to write better dating messages—in 30 seconds, not 30 minutes"

**Reddit/Social:**
"Tired of copying messages to ChatGPT? We built an app for that. Suggestions, coaching, confidence."

**TikTok/Content:**
- "3 ways to write a better opening message"
- "Why your dating messages aren't working"
- "This app makes messaging less stressful"

---

## 13. RISK MITIGATION

### 13.1 Legal Risks

| Risk | Mitigation |
|------|-----------|
| **Platform ToS violation** | No API access, copy-paste only, no automation |
| **Liability for message content** | Clear ToS stating user responsibility |
| **Privacy violations** | Zero storage of messages, analytics only |
| **False positive fraud detection** | Human review before blocking, warning first |
| **Defamation from scam flagging** | Only flag User A, never User B |

### 13.2 Technical Risks

| Risk | Mitigation |
|------|-----------|
| **Clipboard access denied** | Fallback to manual paste |
| **App rejected from stores** | Hire lawyer review before submission |
| **High API costs** | Monitor Claude API usage, optimize prompts |
| **Server downtime** | Local-first design, suggestions cached |
| **Data breach** | Encrypt everything, minimal data stored |

### 13.3 Business Risks

| Risk | Mitigation |
|------|-----------|
| **Low retention** | Focus on value (suggestions must be quality) |
| **User confusion** | Clear UI messaging throughout |
| **Scaling costs** | Charge users fairly, API costs reasonable at scale |
| **Competition** | Move fast, focus on UX, build community |
| **Dating app bans** | Don't violate ToS, respect platform rules |

---

## 14. SUCCESS CRITERIA

### Month 1
- [ ] MVP launched and working
- [ ] 100+ beta users (friends, family)
- [ ] Legal docs reviewed by lawyer
- [ ] Feedback collected and analyzed

### Month 3
- [ ] 1,000+ users
- [ ] Reply coaching feature solid
- [ ] 5-10% freemium conversion rate
- [ ] Browser extension live
- [ ] $500-1,000 MRR

### Month 6
- [ ] 10,000+ users
- [ ] Opening message feature live
- [ ] 5-10% retention (DAU/MAU)
- [ ] $5,000-10,000 MRR
- [ ] Positive user reviews

### Month 12
- [ ] 50,000+ users
- [ ] Dashboard live
- [ ] Fraud detection active
- [ ] $25,000-50,000 MRR
- [ ] Sustainable unit economics

---

## 15. QUICK REFERENCE: DO's & DON'Ts

### DO ✅
- ✅ Store User A's preferences and notes
- ✅ Generate coaching suggestions
- ✅ Help User A write better messages
- ✅ Track User A's app usage for analytics
- ✅ Detect fraud by User A
- ✅ Give feedback on User A's writing
- ✅ Coach on messaging approaches
- ✅ Educate about dating communication
- ✅ Let User A delete their account

### DON'T ❌
- ❌ Store message content
- ❌ Create profiles about User B
- ❌ Track User B's behavior
- ❌ Score or rate User B
- ❌ Analyze User B's psychology
- ❌ Send messages for User A
- ❌ Auto-fill replies as default
- ❌ Require User B notification
- ❌ Store conversation history
- ❌ Share data with third parties

---

## 16. FINAL NOTES FOR DEVELOPERS

### Product Philosophy
**Moly is a coaching tool, not a ghostwriting service.**

Every feature should ask: "Does this help User A communicate better, or does it bypass User A's agency?"

**Good features:**
- Suggestions User A must read and process
- Coaching User A can apply
- Feedback User A can learn from
- Tools User A controls completely

**Bad features:**
- Auto-send
- Copying suggestions directly
- Profiling other users
- Manipulation tactics

### Code Quality Standards
- [ ] All sensitive data encrypted
- [ ] No message storage in code paths
- [ ] All API responses cleaned (no storage)
- [ ] User deletion removes all data
- [ ] Security audit before launch
- [ ] Privacy impact assessment

### Testing Checklist
- [ ] Can User A use app without creating profile?
- [ ] Can User A disable all tracking?
- [ ] Do all suggestions require reading (not copying)?
- [ ] Can User A complete flow without sending via Moly?
- [ ] Does ToS clearly state liability?
- [ ] Can users delete account easily?

---

## APPENDIX A: Example Prompts for Claude API

### Suggestion Generation (Replies)

```
User: You are a dating coach. Generate 2-3 response suggestions 
for this incoming message. Use different tones/approaches.

Focus on authenticity, not manipulation.
Keep responses short (1-2 sentences).
Each suggestion should feel like a real person wrote it.

Incoming message: "{incoming_message}"

Format:
1. [Tone]: [Response]
2. [Tone]: [Response]
3. [Tone]: [Response]
```

### Opening Message Coaching

```
User: You are a dating coach. The user wants to write an opening 
message to someone new. They've told you about this person.

Help them understand different approaches to opening messages. 
Provide examples of approaches, NOT messages to copy.

What they know: "{user_knowledge}"

Suggest 3 approaches with example structures (not exact copies):
1. [Approach]: Example structure: "..."
2. [Approach]: Example structure: "..."
3. [Approach]: Example structure: "..."

Then give coaching: What makes a good opening message?
```

### Feedback on User's Writing

```
User: You are a dating coach. The user wrote this message 
and wants feedback.

Give constructive, encouraging feedback. Point out what's good, 
suggest one improvement.

Message: "{user_message}"

Feedback:
- What's working: ...
- One suggestion: ...
```

---

## APPENDIX B: Conversation with Founders

**Key Decisions Made:**

1. **Copy-paste model:** Safer than API integration, respects platform ToS
2. **No message storage:** Zero breach liability, compliant with privacy laws
3. **Coaching only:** User A writes everything, owns all messages
4. **No User B profiling:** Ethical boundary, legal protection
5. **Manual context:** User A provides info about people (not scraped)
6. **Freemium monetization:** Lower friction, proven SaaS model
7. **Phased rollout:** Browser extension first, then mobile, then features
8. **Fraud detection:** Block bad actors, report to authorities

**Why These Matter:**

These decisions transform Moly from a risky AI startup into a defensible, ethical coaching tool. They also make it clearer to users what they're getting and what they're responsible for.

---

## APPENDIX C: Folder Structure for Developers

```
moly/
├── README.md (this document)
├── docs/
│   ├── LEGAL_FRAMEWORK.md
│   ├── TECHNICAL_SPEC.md
│   ├── API_INTEGRATION.md
│   ├── PRIVACY_POLICY.md (template)
│   └── TERMS_OF_SERVICE.md (template)
├── src/
│   ├── components/
│   │   ├── SuggestionCard.tsx
│   │   ├── OpeningCoach.tsx
│   │   ├── Dashboard.tsx
│   │   └── ConversationAnalytics.tsx
│   ├── services/
│   │   ├── claudeAPI.ts
│   │   ├── clipboard.ts
│   │   ├── storage.ts
│   │   └── analytics.ts
│   ├── types/
│   │   ├── user.ts
│   │   ├── suggestions.ts
│   │   └── analytics.ts
│   └── screens/
│       ├── Home.tsx
│       ├── Reply.tsx
│       ├── Opening.tsx
│       └── Dashboard.tsx
├── tests/
│   ├── suggestions.test.ts
│   ├── storage.test.ts
│   └── privacy.test.ts
├── legal/
│   ├── terms-of-service.md
│   ├── privacy-policy.md
│   └── launch-checklist.md
└── marketing/
    ├── positioning.md
    ├── messaging.md
    └── launch-plan.md
```

---

**Version:** 1.0  
**Last Updated:** August 3, 2026  
**Status:** Ready for Development

**For Questions:** Refer back to conversation thread for full context and reasoning.
