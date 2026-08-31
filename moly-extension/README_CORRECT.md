# Moly - AI Coaching Chatbot

**Version 2.0 (Conversational Architecture)**

An intelligent browser extension that helps you craft better messages through conversation.

---

## What is Moly?

Moly is your **personal AI coach** for messaging.

Instead of just suggesting responses, Moly **talks with you** to understand context, then provides personalized suggestions.

### How it works:
1. You chat with Moly to build context about the person
2. You can paste their messages for more specific advice
3. Moly generates suggestions based on full conversation
4. You copy the suggestion and send it yourself

---

## Quick Start

### 1. Install
- Chrome Web Store (coming soon)
- Or install manually from `dist/`

### 2. Configure
- Click M → Settings
- Enter your LLM API key (Claude, OpenAI, or use Ollama)
- Save

### 3. Use
1. Open Moly sidebar (click M icon)
2. Chat with Moly about the person you're responding to
3. Paste their message when ready
4. Copy the suggestion Moly provides
5. Paste into actual chat

---

## Key Features

✓ **Conversational AI Coach**
- Chat-based interface
- Asks guiding questions (Socratic mode)
- Provides direct suggestions (Direct mode)
- Learns from conversation context

✓ **Multi-LLM Support**
- Claude (Anthropic) - Recommended
- OpenAI (GPT-4, GPT-3.5)
- Ollama (local, maximum privacy)

✓ **Per-User Conversation History**
- Separate conversations for each contact
- Full context maintained
- Export conversations as JSON
- Delete anytime

✓ **Communication Modes**
- **Socratic**: Guiding questions for deeper thinking
- **Direct**: Ready-to-use suggestions

✓ **Communication Contexts**
- **Formal**: Professional tone
- **Friendly**: Casual, warm tone
- **Dating**: Playful, engaging tone

✓ **Complete Privacy**
- All data stored locally
- No servers, no tracking
- Only sends to your chosen LLM
- Delete all data instantly

✓ **Works Everywhere**
- Any website, any messaging platform
- No platform-specific code
- Completely policy-compliant

---

## Why Moly?

### vs. No Tool
- Better suggestions through conversation
- Learn from context
- Improve your communication skills
- Save time thinking

### vs. ChatGPT
- Focused on messaging specifically
- Conversation history per person
- Integrated into your browser
- Optimized for dating, professional, casual

### vs. Auto-Sending Bots
- You have full control
- You review before sending
- No account ban risk
- No platform interference
- 100% compliant

---

## Architecture

### Simple & Clean
```
Browser Extension
├── Sidebar (chat interface)
├── Conversation history (per user, local)
├── LLM provider (Claude, OpenAI, Ollama)
└── Settings (LLM config)

NO Content Scripts
NO DOM reading
NO automatic actions
```

### Why This Design?
- **Compliant**: No policy violations
- **Private**: Local storage only
- **Safe**: User fully in control
- **Smart**: Context-aware suggestions
- **Simple**: Easy to understand and use

---

## Documentation

**Read these in order:**

1. **[REDESIGN_v2_CORRECT.md](REDESIGN_v2_CORRECT.md)** - Architecture & design
2. **[USER_GUIDE_CORRECT.md](docs/USER_GUIDE_CORRECT.md)** - How to use
3. **[COMPLIANCE.md](docs/COMPLIANCE.md)** - Policy compliance
4. **[PRIVACY_POLICY.md](docs/PRIVACY_POLICY.md)** - Privacy & security
5. **[CLAUDE.md](CLAUDE.md)** - Developer guide

---

## Development

### Setup
```bash
npm install
npm run dev    # Development
npm run build  # Production
npm run lint   # Code quality
```

### Tech Stack
- React 18 + TypeScript
- Zustand (state management)
- Tailwind CSS
- Vite
- Chrome Extension API

### Project Structure
```
src/
├── sidebar/          # Main chat interface
├── stores/           # State management
├── api/              # LLM providers
├── popup/            # Extension menu
└── settings/         # Configuration
```

---

## Roadmap

### v2.0 (Current) - MVP
- ✓ Chat interface
- ✓ Conversation history
- ✓ Multi-LLM support
- ✓ Context-aware suggestions
- ✓ Full compliance

### v2.1 (3-4 weeks)
- Contact management UI
- Search history
- Custom templates
- Keyboard shortcuts
- Export conversations

### v2.2 (4-5 weeks)
- Premium tier ($2.99/month)
- Advanced AI features
- Encrypted sync
- Team collaboration

### v3.0 (2-3 months)
- Mobile app
- Desktop client
- Developer API
- Official integrations

---

## Compliance & Privacy

### Fully Compliant ✓
- Facebook Messenger
- Instagram DMs
- Tinder, Hinge, Bumble
- FetLife, Discord, Slack
- Any messaging platform

### Why Compliant?
- No automatic reading
- No DOM manipulation
- No platform APIs used
- User-controlled data
- Completely transparent

### Privacy Guaranteed ✓
- All data stored locally
- No tracking or analytics
- No external communication (except LLM)
- Easy deletion anytime
- GDPR compliant

**See [COMPLIANCE.md](docs/COMPLIANCE.md) for detailed analysis.**

---

## FAQ

**Q: Is Moly free?**  
A: Yes, free tier. Premium in v2.1.

**Q: Will I get banned?**  
A: No. Moly is a coaching tool, not a bot.

**Q: Does Moly send messages?**  
A: No. You copy and send manually.

**Q: Is my data private?**  
A: Completely. All local, nothing shared.

**Q: Does it work on mobile?**  
A: Desktop Chrome only (for now).

**Q: Can I use offline?**  
A: Yes with Ollama, no with Claude/OpenAI.

---

## Support

- **Email**: efthimiosangelopoulos@gmail.com
- **Issues**: GitHub (when open-sourced)
- **Guide**: See [USER_GUIDE_CORRECT.md](docs/USER_GUIDE_CORRECT.md)

---

## Contributing

We're building Moly to help people communicate better while respecting privacy and safety.

**Core values:**
1. Privacy first - data always local
2. Safety - no platform violations
3. Intelligence - context-aware suggestions
4. Simplicity - easy to understand and use
5. Control - users decide everything

---

## License

Moly is proprietary commercial software.

---

## Acknowledgments

- Anthropic (Claude)
- OpenAI (GPT)
- Ollama team (local LLMs)
- Chrome team (extension APIs)

---

**Moly v2: Intelligent Messaging, Complete Privacy** 🎯

*Last Updated: August 31, 2026*  
*Status: Ready for Implementation*
