# Moly v2 Redesign: Conversational Coaching Chatbot

**Version**: 2.0 (Correct Architecture)  
**Date**: 2026-08-31  
**Status**: APPROVED FOR IMPLEMENTATION

---

## Executive Summary

**Moly is a conversational AI coach** that helps users craft better messages through intelligent dialogue.

### How It Works
1. User opens Moly sidebar (chat interface)
2. User has a conversation with Moly to build context
3. User can paste incoming messages for context
4. Moly suggests responses based on conversation history
5. User can copy suggestions or continue chatting for refinement

### Key Differentiators
- **Conversational**: Not just a suggestion tool, but an AI coach
- **Context-Aware**: Learns from chat history with user
- **Local Storage**: All conversation history stored locally per user
- **Privacy-First**: No automatic reading, user-controlled data
- **Fully Compliant**: No platform policy violations
- **Multi-LLM**: Works with Claude, OpenAI, Ollama

---

## Architecture Overview

### User Interaction Flow

```
USER OPENS MOLY SIDEBAR
        ↓
CHAT INTERFACE LOADS
        ↓
USER: "I matched with someone on Tinder"
        ↓
MOLY: "That's exciting! Tell me about them"
        ↓
USER: "They seem fun and genuine"
        ↓
MOLY: "What's your intention? Dating or casual?"
        ↓
USER: "Looking for a real connection"
        ↓
MOLY: "Got it. What communication style do you prefer?
       [Select: Socratic or Direct]"
        ↓
USER: "Direct - I like being ready to respond"
        ↓
USER: [PASTES: "Hey, you look fun!"]
        ↓
MOLY: "Based on our conversation, here are responses:
       1. 'Thanks! You seem interesting too.'
       2. 'I'd love to know more about you!'"
        ↓
USER: [COPIES RESPONSE AND PASTES INTO TINDER]
```

### Architecture Diagram

```
┌─────────────────────────────────────────────┐
│         Moly Sidebar (Chat UI)              │
├─────────────────────────────────────────────┤
│                                             │
│ ┌───────────────────────────────────────┐  │
│ │ CONVERSATION HISTORY                  │  │
│ │ ─────────────────────────────────────  │  │
│ │ Moly: Who are you trying to respond to?│  │
│ │ You: Someone on Tinder                │  │
│ │ Moly: Tell me about them              │  │
│ │ You: They seem fun                    │  │
│ │ Moly: What's your intent?             │  │
│ │ [scroll...]                           │  │
│ └───────────────────────────────────────┘  │
│                                             │
│ ┌───────────────────────────────────────┐  │
│ │ MESSAGE INPUT                         │  │
│ │ [User types or pastes message here]   │  │
│ │ ┌─────────────────────────────────────┤  │
│ │ │ Hey, you look fun!                  │  │
│ │ └─────────────────────────────────────┤  │
│ │ [Paste] [Send] [Clear]                │  │
│ └───────────────────────────────────────┘  │
│                                             │
│ ┌───────────────────────────────────────┐  │
│ │ SUGGESTED RESPONSES                   │  │
│ │ ─────────────────────────────────────  │  │
│ │ 1. "Thanks! You seem interesting too" │  │
│ │    [Copy]                             │  │
│ │ 2. "I'd love to know more about you!" │  │
│ │    [Copy]                             │  │
│ │ 3. "Haha, thanks! What are you up to?"│  │
│ │    [Copy]                             │  │
│ └───────────────────────────────────────┘  │
│                                             │
│ ┌───────────────────────────────────────┐  │
│ │ SETTINGS                              │  │
│ │ Mode: [Socratic ✓] [Direct]           │  │
│ │ Context: [Formal] [Friendly ✓] [Dating│  │
│ │ LLM Provider: [Claude ✓] [OpenAI] [...] │
│ │ [Settings] [Clear History] [Export]   │  │
│ └───────────────────────────────────────┘  │
│                                             │
└─────────────────────────────────────────────┘

LOCAL STORAGE (Chrome Storage)
├── Users
│   ├── User 1 (ID: hash)
│   │   ├── conversationHistory: [...]
│   │   ├── settings: {mode, context, llmProvider}
│   │   └── contacts: [{name: "John", context: "..."}]
│   ├── User 2 (ID: hash)
│   │   └── ...
│   └── User 3 (ID: hash)
│       └── ...

LLM PROVIDER
├── Claude (Anthropic API)
├── OpenAI (GPT-4, GPT-3.5)
└── Ollama (Local)
```

---

## Component Architecture

### Extension Structure

```
moly-extension/
├── public/
│   ├── manifest.json (SIMPLIFIED - no content scripts)
│   ├── icon16.png
│   ├── icon48.png
│   └── icon128.png
│
├── src/
│   ├── popup/
│   │   ├── popup.html (extension menu)
│   │   ├── Popup.tsx (open sidebar, settings)
│   │   └── popup.css
│   │
│   ├── sidebar/
│   │   ├── sidebar.html (main entry point)
│   │   ├── Sidebar.tsx (MAIN COMPONENT - chat interface)
│   │   ├── components/
│   │   │   ├── ChatHistory.tsx (display past messages)
│   │   │   ├── MessageInput.tsx (user input + paste)
│   │   │   ├── Suggestions.tsx (AI suggestions)
│   │   │   ├── Settings.tsx (mode, context, LLM)
│   │   │   └── ContactContext.tsx (who are we talking to)
│   │   └── sidebar.css
│   │
│   ├── settings/
│   │   ├── settings.html
│   │   ├── Settings.tsx (LLM config only)
│   │   └── settings.css
│   │
│   ├── stores/
│   │   ├── chatStore.ts (MAIN STORE - conversation history)
│   │   ├── settingsStore.ts (user preferences, LLM config)
│   │   ├── contactStore.ts (per-contact context)
│   │   └── userStore.ts (user identification)
│   │
│   ├── api/
│   │   ├── providerManager.ts (LLM abstraction)
│   │   ├── providers/
│   │   │   ├── claude.ts (Claude API)
│   │   │   ├── openai.ts (OpenAI API)
│   │   │   └── ollama.ts (Local Ollama)
│   │   ├── memooryManager.ts (conversation context building)
│   │   └── suggestionsEngine.ts (generate responses)
│   │
│   ├── types/
│   │   └── index.ts (TypeScript interfaces)
│   │
│   └── utils/
│       └── clipboard.ts (clipboard operations)
│
└── dist/ (built output)
```

---

## Key Components Explained

### 1. ChatHistory Component
**Purpose**: Display conversation history with Moly

**Displays**:
- Moly's coaching questions
- User's responses and context
- Previous incoming messages (pasted)
- Previous suggestions made

**Features**:
- Scrollable conversation view
- Delete individual message pairs
- Export conversation
- Search within history

### 2. MessageInput Component
**Purpose**: User input for text and pasted messages

**Features**:
- Text area (type or paste)
- "Paste from Clipboard" button
- "Send" button (send to Moly as input)
- "Clear" button (clear input)
- Character counter
- Support for multi-line input

### 3. ContactContext Component
**Purpose**: Track who the user is talking to

**Tracks**:
- Contact name (who are we responding to)
- Platform (Tinder, Facebook, etc.)
- Conversation context (what we know about them)
- User's intent (dating, friendship, etc.)

**Updates**:
- User can change contact anytime
- Context saved per contact
- Moly references this in suggestions

### 4. Suggestions Component
**Purpose**: Display AI-generated response suggestions

**Features**:
- 3-5 suggestions (configurable)
- Each with [Copy] button
- Tone indicator (Socratic/Direct)
- Context indicator (Formal/Friendly/Dating)
- "Ask Moly for more" button
- "Refine suggestion" button (starts new conversation)

### 5. Settings Component (Sidebar)
**Purpose**: Quick settings in sidebar

**Options**:
- Mode: [Socratic] [Direct]
- Context: [Formal] [Friendly] [Dating]
- LLM Provider: [Claude] [OpenAI] [Ollama]
- Link to full settings page

### 6. ConversationStore (NEW - CORE)
**Purpose**: Manage conversation history per user

```typescript
interface Message {
  id: string;
  type: 'user' | 'moly' | 'incoming' | 'suggestion';
  content: string;
  timestamp: number;
  metadata?: {
    mode?: 'socratic' | 'direct';
    context?: 'formal' | 'friendly' | 'dating';
    contact?: string;
  };
}

interface Conversation {
  id: string;
  userId: string;
  contactName?: string;  // Who we're talking to
  contactPlatform?: string;  // Where they contacted from
  messages: Message[];
  settings: {
    mode: 'socratic' | 'direct';
    context: 'formal' | 'friendly' | 'dating';
    llmProvider: 'claude' | 'openai' | 'ollama';
  };
  createdAt: number;
  updatedAt: number;
}

interface ChatStore {
  conversations: Conversation[];
  currentConversation: Conversation | null;
  
  // Actions
  startConversation: (contact?: string) => void;
  loadConversation: (id: string) => void;
  addMessage: (content: string, type: 'user' | 'moly' | 'incoming') => void;
  generateSuggestions: (conversationContext: Message[]) => Promise<string[]>;
  updateSettings: (settings: Partial<Conversation['settings']>) => void;
  deleteConversation: (id: string) => void;
  exportConversation: (id: string) => string; // JSON
}
```

---

## User Workflow (Detailed)

### Scenario: User Matches on Tinder

**Step 1: User Opens Moly**
```
Click M icon → Sidebar opens
Moly: "Hey there! What can I help you with today?"
```

**Step 2: User Provides Context**
```
User: "I just matched with someone on Tinder"
Moly: "Exciting! Tell me a bit about them"
User: "They seem really fun and genuine, good sense of humor"
Moly: "That's great! What are you looking for in this connection?"
User: "I'm interested in dating, maybe something serious"
Moly: "Got it. How would you like me to help you respond?
       Would you prefer direct suggestions or want me to ask 
       guiding questions?"
User: [Selects Direct mode]
```

**Step 3: User Provides Message to Respond To**
```
User: [Pastes incoming message: "Hey, you look fun! 😊"]
Moly: "Based on what you've told me, here are some responses:
       1. Thanks! You seem interesting too. What's your story?
       2. Haha, thanks! What do you like to do for fun?
       3. Appreciate it! Tell me about yourself"
```

**Step 4: User Copies and Sends**
```
User: Clicks [Copy] on suggestion #1
User: Switches to Tinder tab
User: Ctrl+V in chat box
User: Sends message
```

**Step 5: Follow-up (Later)**
```
User: [Comes back to Moly with new incoming message]
User: [Pastes: "I love hiking and coffee, you?"]
Moly: "Based on your previous conversation, here are options:
       1. That sounds amazing! I'm into hiking too, maybe we could 
          hike sometime? As for coffee, I'm obsessed!
       2. Love those things! What's your favorite hiking trail?
       3. That's cool! I like both too. Where do you usually go hiking?"
```

**Key Features**:
- Moly remembers previous context
- Suggestions get better with more conversation
- User can continue chatting with Moly for refinement
- Full conversation history saved locally

---

## Data Storage Strategy

### Per-User Separation
Each user gets their own conversation space:
```
Chrome Storage
└── moly:
    ├── user:current-user-id: "john-doe-123"
    ├── conversations:
    │   ├── conv-001-tinder-match-1
    │   │   ├── contactName: "Sarah"
    │   │   ├── contactPlatform: "Tinder"
    │   │   ├── messages: [...]
    │   │   └── settings: {...}
    │   ├── conv-002-tinder-match-2
    │   └── conv-003-instagram-dm
    ├── settings:global
    │   ├── llmProvider: "claude"
    │   └── apiKey: "sk-..."
    └── contacts:
        ├── sarah-tinder
        ├── jessica-instagram
        └── maya-bumble
```

### Privacy Guarantee
- No data transmitted to Moly servers
- Only to user's selected LLM provider
- User can delete all data anytime
- Each browser profile is isolated

---

## Compliance & Policy

### Why This Architecture is Compliant

**1. No Automatic Reading**
- User manually pastes messages
- No DOM observation
- No content script injection
- Completely explicit

**2. User-Controlled**
- User decides what to share with Moly
- User can delete anytime
- User sees exactly what's stored
- User chooses LLM provider

**3. No Platform Interference**
- Sidebar is separate from actual chat
- Moly never sends messages
- Moly never modifies chat
- User manually copies/pastes

**4. Works Everywhere**
- Not platform-specific
- Works on ANY website
- No platform-specific code
- Generic coaching tool

**Policy Compliance:**
- ✓ Facebook Messenger
- ✓ Instagram DMs
- ✓ Tinder, Hinge, Bumble
- ✓ FetLife, Discord, Slack
- ✓ Any messaging platform

---

## Feature Comparison: v1 vs v2 (Correct)

| Feature | v1 (Auto-Read) | v2 (Chatbot) |
|---------|----------------|-------------|
| **Input Method** | Auto-detect from DOM | User pastes + chat |
| **Interaction** | Passive suggestions | Active conversation |
| **Context Building** | Single message | Full conversation history |
| **Platform Compliance** | ✗ Violates | ✓ Compliant |
| **User Ban Risk** | MEDIUM-HIGH | ZERO |
| **Data Privacy** | Good | EXCELLENT |
| **Suggestion Quality** | Good | EXCELLENT (context-aware) |
| **Works Everywhere** | Only chat sites | All websites |
| **Sellable** | No | Yes |

---

## Technology Stack

### Frontend
- React 18 (component UI)
- TypeScript (type safety)
- Tailwind CSS (styling)
- Zustand (state management - improved for conversations)

### Backend/API
- Claude API (Anthropic)
- OpenAI API (GPT)
- Ollama (local LLM)
- Chrome Storage API (local persistence)

### Build Tools
- Vite (build system)
- TypeScript compiler
- Chrome extension manifest v3

---

## Roadmap

### v2.0 (MVP - This Release)
- ✓ Chat interface with Moly
- ✓ Conversation history per user
- ✓ Context-aware suggestions
- ✓ Socratic/Direct modes
- ✓ Communication contexts
- ✓ Multi-LLM support
- ✓ Local storage per user
- ✓ Full compliance

### v2.1 (3-4 weeks)
- Contact management UI
- Search conversation history
- Export conversations
- Custom prompts/templates
- Keyboard shortcuts

### v2.2 (4-5 weeks)
- Premium tier ($2.99/month)
- Advanced AI features
- Sync across devices (encrypted)
- Team collaboration
- Analytics (local only)

### v3.0 (2-3 months)
- Mobile app (iOS/Android)
- Desktop client
- Developer API
- Official platform integrations
- Advanced customization

---

## Implementation Timeline

### Phase 1: Chat Architecture (6-8 hours)
- Redesign Sidebar for chat interface
- Create ChatHistory component
- Create MessageInput component
- Update stores for conversation management
- Wire up basic chat flow

### Phase 2: Context Management (4-6 hours)
- Create ContactContext component
- Add conversation history tracking
- Implement per-user storage
- Add contact management

### Phase 3: AI Integration (4-6 hours)
- Update suggestion engine to use conversation context
- Implement coaching prompts for Socratic mode
- Implement direct suggestions for Direct mode
- Test with all LLM providers

### Phase 4: UI/UX Polish (4-6 hours)
- Improve chat display
- Add animations
- Improve responsiveness
- Add keyboard shortcuts
- Better error handling

### Phase 5: Testing & Release (4-6 hours)
- Comprehensive testing
- Chrome Web Store submission
- Documentation
- Beta testing
- Final launch

**Total: 22-32 hours**

---

## Success Criteria

- ✓ Chat interface fully functional
- ✓ Conversation history saved locally
- ✓ Multi-LLM support working
- ✓ Context-aware suggestions
- ✓ Full compliance verified
- ✓ Zero user ban risk
- ✓ Chrome Web Store approved
- ✓ Documentation complete
- ✓ Ready for commercial sale

---

## Key Differences from Original v1

**This is NOT an auto-reading tool.**

This is an **AI coaching chatbot** that:
1. Learns through conversation with user
2. Accepts incoming messages for context
3. Provides intelligent, context-aware suggestions
4. Maintains full conversation history per user
5. Works on any website/platform
6. Completely policy-compliant
7. Privacy-first (local storage)
8. Ready for commercial sale

---

**This is the correct, approved architecture.**

All documentation, code, and implementation should follow this specification.
