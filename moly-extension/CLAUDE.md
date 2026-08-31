# Moly v2 Project Instructions

**Project**: Moly - AI Coaching Chatbot Extension  
**Version**: 2.0 (Conversational Architecture)  
**Date**: August 31, 2026  
**Status**: Ready for Implementation  

---

## Project Overview

Moly is a **conversational AI coaching chatbot** that helps users craft better messages through intelligent dialogue.

### Core Value Proposition
- **Conversational**: Chat with AI to build context about the person you're responding to
- **Context-Aware**: Moly remembers conversation history, provides smarter suggestions
- **User-Controlled**: User decides what to share, can paste incoming messages
- **Privacy-First**: All conversation history stored locally per user
- **Policy-Compliant**: No automatic reading, no platform interference
- **Multi-LLM**: Works with Claude, OpenAI, Ollama (local)

### Business Model
- **Immediate**: Free tier on Chrome Web Store
- **Short-term**: Premium features ($2.99/month)
- **Long-term**: Team collaboration, enterprise licensing

---

## Architecture (v2 - Correct)

### User Workflow
```
1. User opens Moly sidebar
2. User explains context to Moly (chat)
3. Moly asks guiding questions (Socratic) or waits for input (Direct)
4. User builds rich context through conversation
5. User pastes incoming message when ready
6. Moly provides suggestions based on full conversation
7. User copies suggestion and sends
```

### Technical Architecture
```
Browser Extension (No Content Scripts)
├── popup.html/Popup.tsx
│   └── Extension menu + settings link
├── sidebar.html/Sidebar.tsx (CHAT INTERFACE)
│   ├── ChatHistory component (conversation display)
│   ├── ContactContext component (who are we talking to)
│   ├── MessageInput component (user chat + paste)
│   ├── Suggestions component (AI-generated responses)
│   └── SettingsPanel component (quick settings)
├── settings.html/Settings.tsx
│   └── LLM provider configuration
├── stores/
│   ├── chatStore.ts (CONVERSATION HISTORY - CORE)
│   ├── settingsStore.ts (user preferences, LLM config)
│   ├── contactStore.ts (per-contact context)
│   └── userStore.ts (user identification)
└── api/
    ├── providerManager.ts (LLM abstraction)
    ├── providers/ (Claude, OpenAI, Ollama)
    ├── memoryManager.ts (conversation context)
    └── suggestionsEngine.ts (response generation)
```

### Key Architectural Points
| Aspect | Details |
|--------|---------|
| **Main Component** | Sidebar chat interface (NOT sidebar with suggestions) |
| **Core Store** | chatStore.ts - manages conversations per user |
| **User Interaction** | Chat-based (user types to Moly) |
| **Context Building** | Through conversation history |
| **Incoming Messages** | User pastes them when ready |
| **Suggestions** | Context-aware, based on full conversation |
| **Storage** | Per-user conversations stored locally |
| **Compliance** | No DOM reading, no platform access |
| **LLM** | Claude, OpenAI, or Ollama |

---

## Feature Specifications

### 1. Text Input Area
**Location**: Top of sidebar  
**Functionality**:
- Text area for message input
- "Paste" button (reads clipboard)
- "Clear" button (reset input)
- Character counter (optional)

**Behavior**:
- User pastes message (Ctrl+V) or uses Paste button
- Triggers automatic analysis once text entered
- Results appear below in real-time

### 2. Mode Selection (Unchanged)
**Socratic Mode**:
- Generates guiding questions
- Helps user develop own response
- Encourages critical thinking
- Tone: Thoughtful, explorative

**Direct Mode**:
- Generates ready-to-use responses
- Quick, natural suggestions
- Faster to implement
- Tone: Conversational, immediate

### 3. Context Selection (Unchanged)
**Formal**:
- Professional, respectful tone
- Proper grammar and punctuation
- Maintains boundaries
- Use for: Business, first dates, authority figures

**Friendly**:
- Warm, casual tone
- Natural language
- Open and welcoming
- Use for: Friends, regular conversations

**Dating**:
- Playful, engaging tone
- Shows genuine interest
- Light humor and warmth
- Use for: Romantic interests, flirting

### 4. Suggestions Display
**Format**:
```
SUGGESTED RESPONSES
1. "Response text here..." [Copy]
2. "Another response..." [Copy]
3. "Third option..." [Copy]
```

**Behavior**:
- 3-5 suggestions per message (configurable)
- Each has [Copy] button
- Clicking [Copy] copies to clipboard
- Visual feedback on copy (briefly shows "Copied!")

### 5. Conversation History
**Storage**:
- Stores user message + suggestions locally
- Accessible in sidebar history panel
- Optional: export as JSON/CSV

**Retention**:
- Default: Keep forever
- Option: Auto-clear after X days
- Manual clear: Settings → Clear All Data

---

## Technical Requirements

### Browser Support
- Chrome 90+ (primary)
- Edge 90+ (secondary)
- Firefox (future)
- Safari (future)

### Permissions Required
```json
{
  "permissions": ["storage", "notifications"],
  "host_permissions": []
}
```

**Note**: No host permissions = no access to websites. Extension runs in isolated context.

### LLM Providers

**Claude (Anthropic)**
- API Key required
- Model: claude-3-sonnet (default)
- Cost: Pay-per-use ($0.003-0.015/1K tokens)
- Quality: Excellent
- Speed: Moderate (2-5s typical)

**OpenAI**
- API Key required
- Model: gpt-4-turbo or gpt-3.5-turbo
- Cost: Pay-per-use ($0.01-0.03/1K tokens)
- Quality: Excellent
- Speed: Fast (1-3s typical)

**Ollama**
- No API key (local)
- Models: Any supported by Ollama
- Cost: Free
- Quality: Depends on model
- Speed: Depends on hardware

---

## Implementation Status

### Completed ✓
- [x] Architecture redesigned for v2
- [x] Policy compliance verified
- [x] Privacy policy created
- [x] User guide created
- [x] Compliance documentation created
- [x] Project instructions documented (this file)

### In Progress (Next Phase)
- [ ] Remove old content script code
- [ ] Simplify manifest.json
- [ ] Redesign sidebar UI
- [ ] Add text input functionality
- [ ] Update stores for manual input
- [ ] Comprehensive testing

### Future
- [ ] Chrome Web Store submission
- [ ] Beta user testing
- [ ] Performance optimization
- [ ] Premium features (v2.1)
- [ ] Mobile support (v3.0)

---

## Code Standards

### TypeScript
- Strict mode enabled
- No `any` types
- Proper typing for all functions
- Interface for all data structures

**Example:**
```typescript
interface Message {
  id: string;
  userInput: string;
  suggestions: string[];
  mode: 'socratic' | 'direct';
  context: 'formal' | 'friendly' | 'dating';
  timestamp: number;
}
```

### React Components
- Functional components only (no class components)
- Hooks for state management
- Props interface for type safety
- Memoization where appropriate

**Example:**
```typescript
interface SidebarProps {
  onSuggest?: (suggestion: string) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ onSuggest }) => {
  // Component logic
};
```

### State Management (Zustand)
- One store per domain (chat, settings, contacts)
- Clear, descriptive action names
- Immutable state updates
- DevTools support enabled

**Example:**
```typescript
const useChatStore = create<ChatState>((set) => ({
  messages: [],
  addMessage: (message) => set((state) => ({
    messages: [...state.messages, message],
  })),
}));
```

### Styling (Tailwind CSS)
- Utility-first approach
- No hardcoded colors (use theme)
- Responsive design (mobile-first)
- Dark mode support planned

---

## File Organization

### Source Files (src/)
```
src/
├── popup/
│   ├── popup.html          (entry point)
│   ├── Popup.tsx           (React component)
│   └── popup.css           (styling)
├── sidebar/
│   ├── sidebar.html        (entry point)
│   ├── Sidebar.tsx         (main UI component)
│   └── sidebar.css         (styling)
├── settings/
│   ├── settings.html       (entry point)
│   ├── Settings.tsx        (configuration UI)
│   └── settings.css        (styling)
├── stores/
│   ├── chatStore.ts        (conversation state)
│   ├── settingsStore.ts    (user preferences)
│   └── contactStore.ts     (deprecated, can remove)
├── api/
│   ├── providerManager.ts  (LLM abstraction layer)
│   └── providers/
│       ├── claude.ts       (Anthropic Claude)
│       ├── openai.ts       (OpenAI GPT)
│       └── ollama.ts       (Local Ollama)
└── types/
    └── index.ts            (TypeScript interfaces)
```

### Documentation (docs/)
```
docs/
├── REDESIGN_v2.md          (this redesign)
├── COMPLIANCE.md           (policy analysis)
├── PRIVACY_POLICY.md       (privacy)
├── USER_GUIDE.md           (user docs)
└── ARCHITECTURE.md         (technical details)
```

### Configuration
```
├── manifest.json           (extension manifest)
├── tsconfig.json           (TypeScript config)
├── vite.config.ts          (build config)
├── tailwind.config.js      (Tailwind config)
└── package.json            (dependencies)
```

---

## Development Workflow

### Local Development
```bash
# Install dependencies
npm install

# Start dev server with hot reload
npm run dev

# Build for production
npm run build

# Type checking
npm run lint

# Run tests (when added)
npm run test
```

### Testing the Extension Locally
1. Build: `npm run build`
2. Go to `chrome://extensions/`
3. Enable "Developer mode"
4. Click "Load unpacked"
5. Select `dist/` folder
6. Test all features
7. Check console (F12) for errors

### Git Workflow
1. Create feature branch: `git checkout -b feature/xyz`
2. Make changes with commits
3. Push to branch: `git push origin feature/xyz`
4. Create PR with description
5. Code review before merge
6. Merge to main/master

---

## Priority Features (MVP)

### Must Have (v2.0)
1. [x] Text input area
2. [x] Manual paste workflow
3. [x] Mode selection (Socratic/Direct)
4. [x] Context selection (Formal/Friendly/Dating)
5. [x] LLM suggestion generation
6. [x] Copy to clipboard buttons
7. [x] Local storage of history
8. [x] Settings page
9. [x] Privacy policy
10. [x] Compliance documentation

### Should Have (v2.1)
- Keyboard shortcuts
- Custom context templates
- Suggestion export
- Search conversation history
- Multiple chat threads

### Nice to Have (v3.0+)
- Premium features
- Cross-device sync
- Custom LLM fine-tuning
- Mobile app
- Team collaboration

---

## Testing Checklist

### Manual Testing
- [ ] Text paste works
- [ ] Clear button works
- [ ] Mode selection updates suggestions
- [ ] Context selection changes tone
- [ ] Copy buttons work
- [ ] Settings save correctly
- [ ] History displays properly
- [ ] Delete history works
- [ ] Extension loads without errors

### Platform Testing
- [ ] Facebook Messenger
- [ ] Instagram DMs
- [ ] Tinder
- [ ] Hinge
- [ ] Bumble
- [ ] FetLife
- [ ] Discord
- [ ] Slack
- [ ] Any random website

### Browser Testing
- [ ] Chrome
- [ ] Chromium
- [ ] Edge
- [ ] Chrome on different OS

### Edge Cases
- [ ] Very long messages (>5000 chars)
- [ ] Special characters
- [ ] Multiple languages
- [ ] Copy without selection
- [ ] Rapid suggestion requests
- [ ] No internet connection (Ollama)
- [ ] Invalid API key
- [ ] Rate limited by LLM

---

## Documentation Updates Needed

- [x] REDESIGN_v2.md (complete)
- [x] COMPLIANCE.md (complete)
- [x] PRIVACY_POLICY.md (complete)
- [x] USER_GUIDE.md (complete)
- [x] README_v2.md (complete)
- [x] CLAUDE.md (this file - complete)
- [ ] ARCHITECTURE.md (technical deep-dive - TODO)
- [ ] API.md (LLM integration docs - TODO)
- [ ] DEVELOPMENT.md (dev setup guide - TODO)

---

## Timeline Estimate

### Phase 1: Code Cleanup (4-6 hours)
- Remove content script code
- Remove message detector
- Remove platform detector
- Simplify manifest.json
- Simplify background script

### Phase 2: UI Redesign (6-8 hours)
- Add text input area
- Add paste button
- Remove auto-detection UI
- Update styling
- Add character counter

### Phase 3: Functionality (4-6 hours)
- Update stores
- Wire up text input
- Update suggestion logic
- Test all features
- Fix bugs

### Phase 4: Testing & Polish (6-8 hours)
- Comprehensive testing
- Edge case handling
- Performance optimization
- Bug fixes
- Documentation updates

### Phase 5: Release Prep (4-6 hours)
- Chrome Web Store assets
- Screenshots and descriptions
- Privacy policy review
- Terms of service
- Final testing

**Total Estimate: 24-34 hours**

---

## Success Criteria

- ✓ No automatic DOM reading
- ✓ Manual copy/paste workflow only
- ✓ Works on any website/platform
- ✓ Fully policy-compliant
- ✓ Zero user ban risk
- ✓ Clear, intuitive UI
- ✓ All features working
- ✓ Comprehensive documentation
- ✓ Chrome Web Store approval
- ✓ Ready for commercial sale

---

## User Feedback Priority

When users report issues, prioritize by:
1. **Critical**: Crashes, data loss, security issues → Fix immediately
2. **High**: Features not working, policy violation risks → Fix within 48 hours
3. **Medium**: UI/UX improvements, documentation → Fix within 1 week
4. **Low**: Enhancement requests, nice-to-haves → Plan for future release

---

## Security Considerations

### No Sensitive Data Collected
- No usernames/passwords
- No session tokens
- No credentials stored
- No payment info
- No PII beyond what user inputs

### No External Communication
- No telemetry
- No analytics
- No crash reporting
- No promotional content
- Only to user's chosen LLM provider

### Browser Security
- Relies on browser's extension sandbox
- No elevated permissions
- Content script isolation
- No eval or dangerous APIs

---

## Future Roadmap

### v2.1 (3-4 weeks after v2.0)
- Keyboard shortcuts
- Custom suggestion templates
- Advanced settings
- Suggestion history search
- Export conversations

### v2.2 (4-5 weeks after v2.1)
- Premium tier ($2.99/month)
- Advanced AI features
- Custom LLM models
- Collaboration features
- Team workspace

### v3.0 (2-3 months after v2.0)
- Mobile app (iOS/Android)
- Desktop client (Electron)
- API for developers
- Cloud sync (encrypted)
- Official platform integrations

---

## Questions to Clarify

1. **Business Model**: Confirm free + premium pricing strategy
2. **Release Timeline**: When to submit to Chrome Web Store?
3. **Marketing**: Any launch campaign planned?
4. **Support**: Email support required?
5. **Localization**: Support other languages?
6. **Accessibility**: WCAG compliance needed?

---

## Contact & Next Steps

**Project Lead**: Efthimios Angelopoulos  
**Email**: efthimiosangelopoulos@gmail.com  

**Next Phase**: Begin implementation based on this specification.

---

## Important Notes

### This is a Complete Redesign
- **v1** (auto-reading) is deprecated and should not be used
- **v2** (manual copy/paste) is the approved architecture
- All development should follow v2 specifications
- v1 code should be removed or archived

### Compliance is Non-Negotiable
- All policy compliance verified
- User privacy is paramount
- No shortcuts on safety
- No compromise on legal risk

### User Experience is Key
- Manual workflow should be as smooth as possible
- Clear instructions throughout
- Helpful error messages
- Responsive UI performance

---

*Last Updated: August 31, 2026*  
*Version: 2.0*  
*Status: Ready for Implementation*
