# Moly Communication Systems

Moly is an AI-powered assistant for ethical written communication across all contexts. This document explains three core systems that help users communicate better, safer, and more ethically.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Mode Transition Analysis](#mode-transition-analysis)
4. [Safety Checker](#safety-checker)
5. [Communication Constitution](#communication-constitution)
6. [Integration](#integration)
7. [API Reference](#api-reference)
8. [Usage Examples](#usage-examples)

---

## Overview

Moly provides three complementary systems for ethical communication:

| System | Purpose | Scope | Risk Focus |
|--------|---------|-------|-----------|
| **Mode Transition Analysis** | Navigate relationship/professional mode changes | Specific relationships/modes | Strategic risks |
| **Safety Checker** | Detect crisis language and illegal activity | Any user message | Immediate harm |
| **Communication Constitution** | Evaluate ethics of communication approach | Any proposed action | Ethical integrity |

### Philosophy

- **Privacy-First:** No storage of concerning messages or full conversations
- **Educational:** Help users think ethically, not enforce rules
- **Transparent:** Clear reasoning for all alerts and recommendations
- **Universal:** Applies to professional, formal, and personal communication
- **Compassionate:** Non-judgmental, supportive approach to improvement

---

## Architecture

### System Interaction Flow

```
User Types Message
        ↓
[Safety Checker] → Detect crisis/illegal → Show resources
        ↓ (safe)
[Message Sent]
        ↓
User Analyzes Mode Transition
        ↓
[Mode Transition Analysis] → Risk assessment + phases
        ↓
[Communication Constitution] → Check recommendation ethics
        ↓
Display constitutional analysis (if enabled)
```

### Data Models

**Three Independent Systems:**
- `safety_check.go` - Crisis/harm detection
- `mode_transition.go` - Strategic guidance for mode changes
- `constitution.go` - Ethical principle evaluation

**Shared Principles:**
- No full message storage
- Metadata-only tracking
- User controls information disclosure
- Metadata used for pattern detection

---

## Mode Transition Analysis

### Purpose
Help users navigate changes in communication mode or relationship dynamic. Provides risk assessment and phase-by-phase guidance.

### Supported Modes
```
Professional ↔ Romantic
Professional ↔ Power Exchange
Friendly ↔ Romantic
Friendly ↔ Power Exchange
Casual ↔ Romantic
Casual ↔ Power Exchange
```

### How It Works

1. **User specifies:**
   - Current mode (how relationship is now)
   - Desired mode (how they want it to become)
   - Optional context (relevant information)

2. **System analyzes:**
   - Risk level for this specific transition
   - Risk factors (workplace, power imbalance, etc.)
   - Mitigation strategies
   - Phase-by-phase approach

3. **Results include:**
   - Overall risk score (0-100)
   - Critical implications
   - 2-3 strategic phases
   - Tactics for each phase
   - Red flags to watch for
   - Critical self-assessment questions
   - Pro/cons analysis

### Example: Professional → Romantic

**Risk Level:** HIGH (75/100)

**Phases:**
1. Break Professional Bubble (2-4 weeks)
   - Share personal interests
   - Suggest casual hangouts
   - Show vulnerability

2. Introduce Romantic Signals (2-3 weeks)
   - Increase eye contact
   - Find one-on-one time
   - Use light humor

3. Express Interest Explicitly (when ready)
   - Private conversation
   - Acknowledge work dynamic
   - Make it safe to decline

**Red Flags:**
- They keep conversations professional
- They mention someone else they're interested in
- They seem uncomfortable with personal questions

### API

**Endpoint:** `POST /api/analyze-mode-shift`

```json
Request: {
  "contact_id": 1,
  "current_mode": "professional",
  "desired_mode": "romantic",
  "context": "We work in different departments"
}

Response: {
  "analysis": {
    "mode_shift_detected": true,
    "risk_level": "high",
    "overall_risk_score": 75,
    "phases": [...],
    "critical_questions": [...],
    "recommendations": [...]
  }
}
```

---

## Safety Checker

### Purpose
Detect crisis language (suicide, self-harm, violence) and illegal activity indicators. Provide immediate crisis resources and decline assistance with illegal requests.

### Detection Categories

**Crisis Indicators:**
- Suicide/self-harm language: "kill myself", "want to die", "overdose"
- Violence intent: "hurt them", "going to attack", "harm someone"
- Extreme distress: "no reason to live", "can't go on"

**Illegal Activity:**
- Drug trafficking/manufacturing
- Robbery/theft
- Human trafficking
- Child exploitation
- Financial fraud
- Extortion/blackmail

### How It Works

1. **User types message**
2. **Safety check runs automatically**
3. **If crisis language detected:**
   - Alert modal appears
   - Shows crisis resources (names, numbers, URLs)
   - Provides recommended actions
   - Message blocked from sending

4. **If illegal activity detected:**
   - Alert declines to assist
   - Suggests legal counsel
   - Non-judgmental tone
   - Message blocked from sending

5. **If safe:**
   - Message proceeds normally
   - No interruption

### Crisis Resources Included

**USA:**
- National Suicide Prevention Lifeline: 988
- Crisis Text Line: Text HOME to 741741

**UK:**
- Samaritans: 116 123

**Australia:**
- Befrienders: 1300 22 4636

**International:**
- IASP directory
- European crisis helplines

### API

**Endpoint:** `POST /api/check-safety`

```json
Request: {
  "text": "I want to hurt myself"
}

Response (Crisis Detected): {
  "alert": {
    "alert_type": "crisis",
    "severity": "immediate",
    "title": "Crisis Support Available",
    "message": "I detected language suggesting...",
    "indicators": ["Expression of intent to harm"],
    "resources": [...],
    "recommendations": [...]
  }
}

Response (Safe): {
  "alert": null
}
```

---

## Communication Constitution

### Purpose
Evaluate whether a proposed communication approach aligns with universal ethical principles. Applies to professional, formal, and personal contexts.

### 10 Principles

**Critical (Foundation):**
1. Honesty & Truthfulness
2. Informed Consent & Clear Agreement
3. Respect for Boundaries
4. Respect for Autonomy & Agency
5. Physical & Emotional Safety

**High (Important):**
6. Clarity & Clear Communication
7. Fairness & Reciprocity
8. Accountability & Follow-Through
9. Transparency About Intentions

**Medium (Contextual):**
10. Context & Power Awareness

### How It Works

1. **User submits action:**
   "Subtly manipulate them into agreeing"

2. **System evaluates against principles:**
   - Honesty: ❌ Violates (manipulation is deceptive)
   - Transparency: ❌ Violates (hidden manipulation)
   - Autonomy: ❌ Violates (removes real choice)
   - Consent: ❌ Violates (not informed)

3. **Results include:**
   - All violations with severity
   - All aligned principles
   - Overall risk level
   - Critical concerns
   - Actionable recommendations

### Risk Levels

- **EXTREME:** Critical principle violated → Don't proceed
- **HIGH:** Multiple high violations → Pause and reconsider
- **MEDIUM:** Some violations → Be cautious
- **SAFE:** No violations → Proceed

### Example: Professional Negotiation

**Action:** "Pressure contractor to accept lower pay by threatening to hire someone else"

**Violations:**
- Fairness & Reciprocity (exploits power imbalance)
- Transparency (hiding that you have other options)
- Respect for Autonomy (using threat instead of negotiation)

**Risk Level:** HIGH

**Recommendations:**
- Negotiate transparently
- Offer genuinely fair terms
- Explain your options clearly
- Allow them to decline without repercussion

### API

**Endpoint:** `POST /api/evaluate-constitution`

```json
Request: {
  "action": "Tell them I love them while hiding my real intentions"
}

Response: {
  "analysis": {
    "analyzed_action": "...",
    "violations": [
      {
        "principle": "Honesty & Truthfulness",
        "severity": "critical",
        "reasoning": "This involves deception..."
      }
    ],
    "overall_risk_level": "EXTREME",
    "is_constitutional": false,
    "recommendations": [
      "DO NOT proceed with this action",
      "Reconsider your approach fundamentally",
      "Prioritize honesty, consent, and respect"
    ]
  }
}
```

**Endpoint:** `GET /api/constitution-principles`

```json
Response: {
  "supreme_principle": "Communicate with honesty, respect, and integrity...",
  "principles": [
    {
      "id": "honesty",
      "name": "Honesty & Truthfulness",
      "severity": "critical",
      "description": "...",
      "questions": [
        "Are you being truthful?",
        "Are you hiding important information?",
        ...
      ]
    }
  ]
}
```

---

## Integration

### How Systems Work Together

**Scenario: User wants to rush mode transition**

```
User: "I want to speed up the intimate power exchange"

1. Mode Transition Analysis:
   - Detects: Casual → Power Exchange
   - Risk: MEDIUM
   - Recommendation: "Go slowly, extensive negotiation needed"

2. Constitution Check (Optional):
   - Evaluates: "Rushing into power exchange without proper negotiation"
   - Result: HIGH risk (consent, safety violated)
   - Recommendation: "Slow down, negotiate explicitly"

3. Combined Guidance:
   - Mode analysis: Here are the phases
   - Constitution: This recommendation is ethical/unethical
   - User makes informed decision
```

### Design Principles

1. **Layered Analysis:**
   - Safety: Immediate harm prevention
   - Mode: Strategic risk assessment
   - Constitution: Ethical integrity check

2. **No Duplication:**
   - Each system has specific purpose
   - Systems don't overlap
   - Combined provide full picture

3. **User Control:**
   - User decides which systems to use
   - Can enable/disable optionally
   - All data user-controlled

---

## API Reference

### Base URL
```
http://localhost:11436/api
```

### Safety Checker

**POST /api/check-safety**
- Checks message for crisis/illegal language
- Blocks unsafe messages
- Shows resources if triggered
- Params: `{text: string}`

### Mode Transition Analysis

**POST /api/analyze-mode-shift**
- Analyzes relationship/professional mode change
- Provides risk assessment and phasing
- Params: `{contact_id, current_mode, desired_mode, context?}`

### Communication Constitution

**POST /api/evaluate-constitution**
- Evaluates action against ethical principles
- Provides constitutional analysis
- Params: `{action: string}`

**GET /api/constitution-principles**
- Returns all principles and supreme principle
- No parameters needed

---

## Usage Examples

### Example 1: Professional Negotiation

```
User: "I'm negotiating salary with a new job offer"

Step 1: Mode Transition Analysis
- No mode change, skip

Step 2: Constitution Check
- Action: "Accept their offer without negotiating"
- Result: MEDIUM risk
- Violations: Fairness & Reciprocity (not advocating for yourself)
- Recommendation: "Negotiate transparently, state your needs"

Step 3: User Action
- Prepares negotiation with confidence
- Knows it's ethical to advocate for fair compensation
```

### Example 2: Mode Change in Relationship

```
User: "I want to transition from friendship to romantic relationship"

Step 1: Constitution Check (Proactive)
- Action: "Love-bomb them with intense affection"
- Result: EXTREME risk
- Violations: Authenticity, Consent, Transparency (hidden manipulation)
- Recommendation: "Be genuine, go slowly, get clear consent"

Step 2: Mode Transition Analysis
- Current: Friendly
- Desired: Romantic
- Risk: MEDIUM
- Phases: 
  1. Increase emotional intimacy
  2. Introduce romantic signals
  3. Express feelings explicitly

Step 3: Constitutional Check on Recommended Tactics
- Tactic: "Share personal vulnerabilities"
- Result: SAFE (builds genuine connection)
- Tactic: "Test their interest before expressing feelings"
- Result: HIGH risk (manipulative)

Step 4: User Action
- Follows recommended phases
- Avoids manipulative tactics
- Communicates authentically
```

### Example 3: Crisis Intervention

```
User: "I want to send them a message saying I can't live like this anymore"

Step 1: Safety Check
- Crisis language detected: "can't live like this"
- Alert triggered: Crisis Support Modal
- Resources displayed: 988, Crisis Text Line, etc.
- Message blocked from sending

Step 2: User Response
- Reviews crisis resources
- Reaches out to support
- Talk to trusted person
- Dismisses alert

Result: User gets help, message not sent, user safety prioritized
```

---

## Deployment

### Installation
```bash
cd moly-go
go build -o moly .
```

### Running
```bash
./moly
# Server starts on localhost:11436
# Sidebar available at http://localhost:11436/sidebar.html
```

### Endpoints Available
- `/api/status` - Server health
- `/api/check-safety` - Safety checking
- `/api/analyze-mode-shift` - Mode transition analysis
- `/api/evaluate-constitution` - Constitutional evaluation
- `/api/constitution-principles` - Get all principles
- `/sidebar.html` - Web interface

---

## System Requirements

- Go 1.16+
- Local or cloud LLM (Ollama, Claude, OpenAI)
- SQLite (included with Go)
- Modern web browser

---

## Privacy & Security

### What Moly Stores
- Contact metadata (name, relationship type, last interaction)
- Interaction summaries (topic, sentiment, date)
- NO full message content
- NO sensitive personal details

### What Moly Doesn't Store
- Complete conversations
- Alert messages
- Crisis language
- Illegal activity reports
- Anything flagged as unsafe

### Encryption
- Local SQLite database
- Optional: AES-256-GCM for API keys
- All sensitive data encrypted

---

## Future Roadmap

1. **Enhanced Detection:**
   - LLM-based violation detection (more nuanced)
   - Behavioral pattern recognition
   - Contextual sensitivity

2. **Interactive Guidance:**
   - Socratic dialogue for ethical reasoning
   - Ask clarifying questions
   - Help users work through decisions

3. **Communication Templates:**
   - Suggest ethical alternatives
   - Rephrase unethical approaches
   - Example communications

4. **Learning & Personalization:**
   - Track helpful recommendations
   - Improve suggestions over time
   - User communication style adaptation

5. **Extended Principles:**
   - Substance abuse support resources
   - Domestic violence safety planning
   - LGBTQ+ affirming resources
   - Grief and loss support

---

## Support & Contribution

### Issues & Feedback
- Report bugs on GitHub
- Suggest features via discussions
- Share use cases and examples

### Contributing
- Fork the repository
- Create feature branches
- Submit pull requests with tests
- Follow existing code style

### Documentation
- README files in each module
- Inline code comments
- API documentation
- Usage examples

---

## License

MIT License - See LICENSE file

---

## Credits

**Inspired by:**
- Socratic-morality (Constitutional AI framework)
- Socrates AI (Multi-agent systems)
- Consent and ethics frameworks
- Crisis support best practices

**Built with:**
- Go (backend)
- JavaScript (frontend)
- SQLite (database)
- Claude AI (LLM)

---

## Contact

For questions, suggestions, or collaborations:
- GitHub Issues: [Moly Repository]
- Discussions: [Moly Discussions]

---

**Last Updated:** September 4, 2026
**Version:** 1.0
**Status:** Production Ready
