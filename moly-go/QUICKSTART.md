# Moly Quick Start Guide

**Version:** 1.0  
**Last Updated:** September 4, 2026

---

## 30-Second Overview

Moly is an AI-powered ethical communication assistant with three core systems:

1. **Mode Transition Analysis** - Navigate relationship/professional changes strategically
2. **Safety Checker** - Detect crisis language and get immediate resources
3. **Communication Constitution** - Evaluate if your approach aligns with ethical principles

---

## Installation & Setup

### Requirements
- Go 1.16 or later
- macOS, Linux, or Windows
- Optional: Ollama (for local AI) or Claude/OpenAI API key

### Step 1: Clone & Build

```bash
git clone https://github.com/Nireus79/Moly.git
cd Moly/moly-go
go build -o moly .
```

### Step 2: Run Moly

```bash
./moly
```

Server starts on `localhost:11436`

### Step 3: Open in Browser

```
http://localhost:11436/sidebar.html
```

---

## First Time Setup

### 1. Choose an AI Provider

**Option A: Local (Recommended for Privacy)**
- Install Ollama: https://ollama.ai
- Download a model: `ollama pull mistral`
- Moly will auto-detect Ollama
- Click "Local" in settings

**Option B: Claude (Anthropic)**
- Get API key at https://console.anthropic.com
- Paste key in Moly settings
- Select Claude model
- Click "Claude" in settings

**Option C: OpenAI**
- Get API key at https://platform.openai.com
- Paste key in Moly settings
- Select GPT model
- Click "OpenAI" in settings

### 2. Create Your First Contact

Click **"+ New"** button
- Name: Who are you reaching out to?
- Platform: Email, WhatsApp, Slack, etc.
- Relationship: Colleague, friend, mentor, etc.

### 3. Start Using Moly

Type in the message box and start chatting!

---

## Using the Three Systems

### System 1: Mode Transition Analysis

**When to use:** Planning a change in how you relate to someone

**Example:** "I want to change from friendship to romantic relationship"

**Steps:**
1. Select contact
2. Click **"Analyze Mode"** (purple button)
3. Choose current mode (e.g., "Friendly")
4. Choose desired mode (e.g., "Romantic")
5. Click "Analyze Transition"

**You'll get:**
- Risk level (0-100 score)
- 2-3 strategic phases
- Tactics for each phase
- Red flags to watch
- Critical questions to ask yourself

---

### System 2: Safety Checker

**When to use:** Automatic - runs on every message

**What it does:**
- Detects crisis language (suicide, self-harm, violence)
- Detects illegal activity (drugs, trafficking, abuse)
- Shows resources if triggered
- Blocks unsafe messages from sending

**Crisis Resources Available:**
- National Suicide Prevention Lifeline: 988 (US)
- Crisis Text Line: Text HOME to 741741 (US)
- Samaritans: 116 123 (UK)
- Befrienders: 1300 22 4636 (Australia)
- International Association for Suicide Prevention

---

### System 3: Communication Constitution

**When to use:** Want to check if your approach is ethical

**Example:** "I'm thinking of love-bombing them to get their interest"

**Steps:**
1. In settings, find "Constitutional Evaluation"
2. Enter your proposed action
3. Click "Evaluate"

**You'll get:**
- Violations (with severity: critical, high, medium)
- Aligned principles
- Overall risk level
- Specific recommendations

**10 Principles Evaluated:**
- Honesty & Truthfulness (critical)
- Informed Consent (critical)
- Respect for Boundaries (critical)
- Respect for Autonomy (critical)
- Physical & Emotional Safety (critical)
- Clear Communication (high)
- Fairness & Reciprocity (high)
- Accountability & Follow-Through (high)
- Transparency About Intentions (high)
- Context & Power Awareness (medium)

---

## Common Workflows

### Workflow 1: Drafting a Difficult Message

```
1. Select contact
2. Click "✎ Draft" (green button)
3. Describe intention: "Invite them to dinner"
4. Paste your draft message
5. Click "Get Suggestions"
6. Review Moly's improvements
7. Adapt and send
```

### Workflow 2: Understanding a Relationship

```
1. Select contact
2. Chat with Moly about the person
3. Moly asks clarifying questions
4. Answer in message box
5. Moly extracts patterns and insights
6. View analytics for this contact
```

### Workflow 3: Planning a Mode Change

```
1. Select contact
2. Click "Analyze Mode"
3. Specify current and desired modes
4. Add context about the situation
5. Get risk assessment and phases
6. Review recommendations
7. Follow phase-by-phase guidance
```

### Workflow 4: Safety Check

```
1. Type a message
2. Safety Checker runs automatically
3. If crisis language detected:
   - Alert modal appears
   - Shows crisis resources
   - Message blocked from sending
4. Click resources to get help
```

---

## Understanding the Dashboard

### Contact Selector
- **Dropdown:** Choose who to talk about
- **"+ New":** Create new contact
- **"✎ Draft":** Get help writing messages
- **"Analyze Mode":** Plan mode transitions
- **"⋮":** Delete contact

### Contact Info Panel
- **About:** What you know about this person
- **Last Interaction:** When you last spoke
- **Recent Topics:** What you talk about

### Message Area
- Shows conversation with Moly
- Displays clarifying questions
- Shows Moly's insights and guidance

### Input Box
- Type messages or answers to questions
- Press Enter to send
- "Send" button to submit
- "Clear" to start fresh

---

## Tips & Tricks

### Tip 1: Get Better Guidance
- Be specific about what you want
- Share context about the relationship
- Answer Moly's questions fully
- Update contact info as you learn more

### Tip 2: Trust the Systems
- Safety Checker protects you automatically
- Mode Analysis is evidence-based
- Constitution checks real ethical principles
- You always make final decisions

### Tip 3: Use Multiple Systems
- Get mode analysis for transitions
- Check constitutional ethics for your approach
- Draft messages for wording help
- Track patterns over time

### Tip 4: Privacy Settings
- Choose local Ollama for maximum privacy
- Data stays on your computer
- No messages stored (metadata only)
- No external LLM calls for analysis

---

## Troubleshooting

### "Server not responding"
```bash
cd moly-go
./moly
```
Make sure server is running at localhost:11436

### "No models available"
- Install Ollama: https://ollama.ai
- Run: `ollama pull mistral`
- Refresh Moly settings

### "API key invalid"
- Check key in your provider's dashboard
- Paste key exactly (no spaces)
- Click save in settings

### "Message blocked - crisis language"
- This is the Safety Checker working
- Click the resources in the alert
- If real emergency: call 911

### "Contact not found"
- Click "Refresh" button
- Create new contact with "+ New"
- Check contact was actually created

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Enter | Send message |
| Shift+Enter | New line |
| Ctrl+K | Open settings (future) |
| Ctrl+N | New contact (future) |

---

## Default Settings

| Setting | Default | Change In |
|---------|---------|-----------|
| Provider | Ollama (local) | Settings |
| Model | mistral | Settings |
| Mode | Direct | Settings |
| Auto-start | Disabled | Settings |

---

## File Locations

| File | Location |
|------|----------|
| Database | ~/.config/moly/moly.db |
| Settings | Settings UI |
| Logs | Terminal output |

---

## Common Questions

**Q: Is my data private?**  
A: Yes. Only metadata is stored locally. No full messages or sensitive content stored.

**Q: Can I use without internet?**  
A: Yes, if using local Ollama. API providers require internet.

**Q: Can multiple people use Moly?**  
A: Each person should run their own instance for privacy.

**Q: What if I make a mistake?**  
A: Moly provides guidance only - you always decide. You can delete contacts or clear messages.

**Q: How much data can it handle?**  
A: 1000+ contacts, 10,000+ interactions without issues.

**Q: Is there a mobile app?**  
A: Not yet. Use web interface on mobile browser for now.

---

## Next Steps

1. **Explore the Three Systems**
   - Try Mode Transition Analysis
   - Test Safety Checker
   - Evaluate some communications

2. **Create Contacts**
   - Add people you interact with
   - Fill in relationship info
   - Add notes about them

3. **Start Conversations**
   - Chat with Moly about your communications
   - Ask for advice on tricky situations
   - Draft important messages

4. **Track Patterns**
   - Review interaction history
   - Look at analytics
   - Improve over time

---

## Getting Help

**Documentation:**
- README_SYSTEMS.md - Complete API reference
- ARCHITECTURE.md - System design details
- ROADMAP.md - Future features

**Issues:**
- Report bugs on GitHub
- Suggest features
- Ask questions

**Contributing:**
- See ROADMAP.md for priorities
- Submit pull requests
- Help improve documentation

---

## What's New in v1.0

✨ **Three Production-Ready Systems:**
- Mode Transition Analysis (6 modes, 6+ transitions)
- Safety Checker (crisis + illegal detection)
- Communication Constitution (10 ethical principles)

✨ **Features:**
- Contact management
- Interaction tracking
- Message drafting assistance
- Context analysis
- Risk assessment

✨ **Privacy:**
- Metadata-only storage
- No message archival
- Encrypted keys
- Local processing

---

## Version Info

- **Current Version:** 1.0 (Production Ready)
- **Build Date:** September 4, 2026
- **Repository:** https://github.com/Nireus79/Moly
- **Go Version:** 1.16+

---

**Happy communicating! Questions? See README_SYSTEMS.md or GitHub Issues.**
