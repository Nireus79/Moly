# Moly User Guide - Conversational Coaching

**Version**: 2.0 (Chat-Based Model)  
**Updated**: August 31, 2026

---

## Welcome to Moly

Moly is your **personal AI coach** that helps you craft better messages through intelligent conversation.

Instead of just giving suggestions, Moly talks with you to understand context, then provides personalized response ideas.

---

## Getting Started

### 1. Install Moly
1. Go to Chrome Web Store
2. Search "Moly - Messaging Coach"
3. Click "Add to Chrome"
4. Accept permissions

### 2. Configure LLM Provider
1. Click M icon → Settings
2. Choose LLM:
   - **Claude** (recommended - best quality)
   - **OpenAI** (fast, reliable)
   - **Ollama** (local, maximum privacy)
3. Enter API key (get from provider's website)
4. Click "Save & Validate"
5. Done!

---

## Basic Workflow

### Step 1: Open Moly
```
Click M icon in Chrome toolbar
↓
Moly sidebar opens on right side
↓
Chat interface ready
```

### Step 2: Start a Conversation
```
Moly: "Hey there! Who are you trying to respond to?"

You: "Someone I matched with on Tinder"

Moly: "That's great! Tell me about them. 
       What's your first impression?"

You: "They seem really fun and have a good sense of humor"

Moly: "Nice! What are you looking for in this connection?"

You: "I'm interested in dating, maybe something serious"
```

### Step 3: Set Your Preferences
```
Moly: "Got it. How would you like me to help?

[Socratic] - I'll ask guiding questions to help you think through your response

[Direct] - I'll give you ready-to-use suggestions"

You: [Click "Direct"]

Moly: "Perfect. I'll provide direct suggestions based on 
       what you've told me."
```

### Step 4: Paste Incoming Message
```
You: [Copy message from their chat]
     "Hey, you look fun! 😊"

You: [Paste into Moly]

Moly: "Based on our conversation, here are some responses:

1. Thanks! You seem interesting too. What's your story?
   [Copy]

2. Haha, thanks! What do you like to do for fun?
   [Copy]

3. Appreciate it! Tell me about yourself
   [Copy]"
```

### Step 5: Use the Suggestion
```
You: [Click [Copy] on suggestion #1]
     (Text copied to clipboard)

You: [Switch to Tinder tab]

You: [Ctrl+V in chat box]

You: [Send normally]
```

### Step 6: Continue Conversation
```
Later, they respond: "I love hiking and coffee, you?"

You: [Paste new message into Moly]

Moly: (Remembers previous context)
      "Here are updated suggestions:
       
1. That sounds amazing! I'm into hiking too, maybe we could 
   hike sometime? As for coffee, I'm obsessed!

2. Love those things! What's your favorite hiking trail?

3. That's cool! I like both too. Where do you usually go hiking?"
```

---

## Communication Modes

### Socratic Mode
**What it does**: Moly asks thoughtful questions to help you develop your own response

**Best for**: 
- When you want a personalized response
- Learning how to communicate better
- Complex situations needing reflection

**Example**:
```
You: "I'm not sure how to respond to their message"

Moly: "What was the tone of their message? 
       Were they being playful or serious?
       
       And what's your current mood? 
       Do you want to match their energy or set a different tone?"

You: "They were playful, and I want to be fun too"

Moly: "Great! So playful and fun. 
       What could you say that shows personality 
       while keeping things light?"

You: "Maybe something with humor?"

Moly: "Exactly! Here are some playful options:
       1. [suggestion 1]
       2. [suggestion 2]
       3. [suggestion 3]"
```

### Direct Mode
**What it does**: Moly provides ready-to-use response suggestions immediately

**Best for**:
- Quick, confident responses
- Times when you're busy
- Similar situations you know how to handle

**Example**:
```
You: [Paste: "Hey, how's it going?"]

Moly: "Based on your profile, here are my suggestions:

1. Pretty good! How about you?
2. Can't complain! What's new with you?
3. All good here! What's up?"

You: [Pick one and copy]
```

---

## Communication Contexts

### Formal
**Use for**: Professional, new connections, or respectful tone
- Proper grammar
- Thoughtful, measured responses
- Professional boundaries

### Friendly
**Use for**: Friends, casual conversations
- Warm tone
- Natural language
- Open and welcoming

### Dating
**Use for**: Romantic interests, flirting
- Playful tone
- Shows genuine interest
- Light humor and warmth

---

## Features Explained

### Conversation History
Moly keeps a complete history of your chats.

**What it includes**:
- Your messages to Moly
- Moly's questions and suggestions
- Incoming messages you pasted
- Settings you selected
- All stored locally

**Benefits**:
- Moly gets better suggestions (more context)
- You can review past interactions
- Switch between conversations
- Search past messages

### Per-User Conversations
Each person gets their own conversation thread.

**You can have**:
- Sarah (Tinder match)
- Jessica (Instagram DM)
- Maya (Bumble date)
- John (friend from college)

Each with separate history.

### Exporting Conversations
Want to save your chat? 

```
Sidebar menu → Export Conversation
↓
Save as JSON file
↓
Keep for your records
```

---

## Tips & Tricks

### Tip 1: Build Rich Context
The more you tell Moly, the better suggestions.

Good:
```
You: "They matched with me on Tinder"
```

Better:
```
You: "They matched with me on Tinder. 
      They seem fun and genuine, good sense of humor.
      I'm interested in dating, looking for something real.
      They're from my city and work in tech."
```

### Tip 2: Paste Relevant Parts
You don't need to paste the whole conversation.

Good:
```
[Paste their latest message]
```

Better:
```
You: "Context: We've been chatting about travel.
      They just said: [paste message]"
```

### Tip 3: Refine Suggestions
Don't like the suggestions? Ask Moly to refine.

```
You: "Can you make these more playful?"

Moly: "Sure! Here are more playful options:
       1. [updated suggestion]
       2. [updated suggestion]
       3. [updated suggestion]"
```

### Tip 4: Ask Follow-up Questions
Chat with Moly, don't just paste messages.

```
You: "They mentioned hiking, but I don't hike. 
      How should I respond to that?"

Moly: "Interesting! Do you want to be honest about not hiking,
       or would you like to suggest an alternative activity?
       
       What other interests do you share?"
```

### Tip 5: Switch Modes Anytime
Change from Socratic to Direct (or vice versa).

```
[In Direct mode, not getting good suggestions?]

You: [Switch to Socratic]

Moly: "Great! Let me ask some questions to help us think this through..."
```

---

## Settings

### LLM Provider
Choose which AI powers Moly:

**Claude (Recommended)**
- Best natural language understanding
- Thoughtful, nuanced suggestions
- $0.003-0.015 per 1K tokens

**OpenAI (GPT-4 or GPT-3.5)**
- Fast, reliable
- Good suggestions
- $0.01-0.03 per 1K tokens

**Ollama (Local)**
- Runs on your computer
- Maximum privacy
- Free
- Requires setup

### Quick Settings (In Sidebar)
- **Mode**: Socratic or Direct
- **Context**: Formal, Friendly, or Dating
- **LLM Provider**: Choose active provider

### Full Settings Page
- LLM configuration
- API key management
- Clear conversation history
- Export conversations
- Privacy settings

---

## Privacy & Security

### What Moly Stores
- Conversation history (locally)
- Pasted messages (locally)
- Settings and preferences (locally)
- Contact information (locally)

### What Moly Does NOT Store
- Platform credentials
- Payment information
- Your identity
- IP address or tracking data

### Data Deletion
Clear all data anytime:
```
Settings → Advanced → Clear All Data
↓
Instantly deleted
```

---

## Troubleshooting

### "No suggestions appearing"
1. Check API key is valid (Settings)
2. Check internet connection
3. Try different LLM provider
4. Check that you're in Direct mode

### "Suggestions seem generic"
1. Give Moly more context (chat with them first)
2. Try Socratic mode instead
3. Change communication context
4. Try Claude provider (better quality)

### "Can't paste from clipboard"
1. Make sure you've copied something first (Ctrl+C)
2. Check that Moly has clipboard permission
3. Try typing instead of pasting

### "Extension not responding"
1. Reload extension (chrome://extensions, click reload)
2. Restart browser
3. Check Chrome console for errors (F12)
4. Reinstall if issues persist

### "Lost my conversation history"
1. Did you clear data? (Cannot recover)
2. Check if browser cache was cleared
3. Make sure you're in same browser profile

---

## FAQ

**Q: Does Moly send messages for me?**  
A: No. Moly only suggests responses. You manually copy and send.

**Q: Can Moly read my messages automatically?**  
A: No. You must manually paste messages you want to respond to.

**Q: Is my data private?**  
A: Yes. All data stays on your device. Nothing sent to Moly servers.

**Q: Will I get banned using Moly?**  
A: No. Moly is a coaching tool, not a bot. Completely safe.

**Q: Does Moly work on mobile?**  
A: Desktop Chrome only. Mobile not supported (yet).

**Q: Can I use Moly offline?**  
A: Only with Ollama (local LLM). Cloud providers need internet.

**Q: How much does it cost?**  
A: Free tier available. Premium coming in v2.1.

**Q: Can I export my conversations?**  
A: Yes. Settings → Export Conversation (JSON format).

**Q: What if I delete Moly?**  
A: All local data deleted. Can reinstall anytime.

**Q: Does Moly work on all dating apps?**  
A: Yes! Works on Tinder, Hinge, Bumble, Bumble, FetLife, etc.

**Q: Can multiple people use Moly on same computer?**  
A: Yes. Each Chrome profile gets separate conversations.

---

## Support

### Need Help?
- Email: efthimiosangelopoulos@gmail.com
- This guide: Bookmark it!
- Chrome Web Store reviews: See Q&A section

### Report a Bug
1. Note what happened
2. Include screenshot if helpful
3. Mention your LLM provider
4. Email to support

### Feature Request
Tell us what would help you!
- Email your idea
- Explain the use case
- Describe the benefit

---

## Conclusion

Moly is your personal messaging coach that respects your privacy and empowers your communication.

**Remember:**
- Build context through conversation
- Moly learns from what you share
- You always have full control
- Your data stays local
- Completely safe and compliant

**Happy messaging!** 🎯

---

*Last Updated: August 31, 2026*  
*Version: 2.0*  
*Status: Ready to Use*
