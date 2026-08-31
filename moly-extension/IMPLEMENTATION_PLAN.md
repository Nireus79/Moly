# Moly v2 Implementation Plan

**Date**: September 1, 2026  
**Status**: READY TO BEGIN  
**Estimated Duration**: 24-32 hours  

---

## Code Audit Results

### Current Structure (35 TypeScript files)

**TO DELETE** (5 files - old DOM reading code)
```
src/content/
  ├── contentScript.ts (10KB)
  ├── messageDetector.ts (6KB)
  ├── messageExtractor.ts (5KB)
  ├── platformDetector.ts (5KB)
  └── messageDetector.test.ts (5KB)
```

**TO REDESIGN** (4 files - sidebar + stores)
```
src/sidebar/
  ├── Sidebar.tsx (10KB) - MAJOR REDESIGN
  ├── index.ts
  ├── sidebar.css (7KB)
  └── sidebar.html

src/stores/
  ├── chatStore.ts (1.5KB) - UPDATE
  ├── conversationStore.ts (3.4KB) - REDESIGN
  ├── contactStore.ts (3.9KB) - REDESIGN
  └── settingsStore.ts (5KB) - MINIMAL CHANGE
```

**TO SIMPLIFY** (1 file)
```
src/background/
  └── serviceWorker.ts - SIMPLIFY (remove DOM handlers)
```

**TO UPDATE** (3 files)
```
src/popup/
  ├── Popup.tsx - UPDATE (remove auto-detect)
  └── PopupEnhanced.tsx - KEEP or DELETE
  
src/settings/
  └── Settings.tsx - MINIMAL CHANGE
```

**TO KEEP** (good foundation)
```
src/api/
  ├── providerManager.ts ✓ (reuse)
  ├── providers/
  │   ├── claude.ts ✓
  │   ├── openai.ts ✓
  │   └── ollama.ts ✓
  ├── chatModes.ts ✓
  ├── llm.ts ✓
  └── modeAwareLLM.ts ✓
```

---

## Implementation Phases

### Phase 1: Code Cleanup (6-8 hours)

**Task 1.1: Delete src/content/** (1 hour)
- [ ] Delete `src/content/contentScript.ts`
- [ ] Delete `src/content/messageDetector.ts`
- [ ] Delete `src/content/messageExtractor.ts`
- [ ] Delete `src/content/platformDetector.ts`
- [ ] Delete `src/content/messageDetector.test.ts`
- [ ] Delete entire `src/content/` directory
- [ ] Verify no imports reference these files

**Task 1.2: Simplify manifest.json** (30 min)
- [ ] Remove `content_scripts` section
- [ ] Remove `host_permissions` section
- [ ] Remove `web_accessible_resources` section
- [ ] Verify manifest validates
- [ ] Test extension loads

**Task 1.3: Simplify serviceWorker.ts** (1 hour)
- [ ] Remove message handlers for `SHOW_MOLY_SIDEBAR`
- [ ] Remove message handlers for `NEW_MESSAGE_DETECTED`
- [ ] Keep only: init, error handling, basic setup
- [ ] Remove any references to `contentScript`
- [ ] Test background script loads

**Task 1.4: Fix imports across project** (1 hour)
- [ ] Search for imports from `src/content/`
- [ ] Search for references to `messageDetector`
- [ ] Search for references to `platformDetector`
- [ ] Search for references to `messageExtractor`
- [ ] Remove/update all references
- [ ] Run TypeScript compiler, fix errors

**Task 1.5: Test cleanup** (1 hour)
- [ ] Build project: `npm run build`
- [ ] Check for TypeScript errors
- [ ] Verify dist/ folder is valid
- [ ] Load extension in Chrome
- [ ] Verify no console errors

---

### Phase 2: Sidebar Redesign - Component Structure (6-8 hours)

**Task 2.1: Create ChatHistory component** (1.5 hours)
```typescript
// src/sidebar/components/ChatHistory.tsx
interface ChatHistoryProps {
  messages: Message[];
  onDeleteMessage: (id: string) => void;
  onExport: () => void;
}

export const ChatHistory: React.FC<ChatHistoryProps> = ({ messages, onDeleteMessage, onExport }) => {
  // Display conversation history
  // User messages on left, Moly messages on right
  // Incoming messages styled differently
  // Suggestions styled differently
}
```

- [ ] Create component file
- [ ] Define Message interface
- [ ] Implement message rendering
- [ ] Add delete functionality
- [ ] Add export button
- [ ] Style for chat UI

**Task 2.2: Create MessageInput component** (1.5 hours)
```typescript
// src/sidebar/components/MessageInput.tsx
interface MessageInputProps {
  onSend: (text: string) => void;
  onPaste: () => void;
  disabled?: boolean;
}

export const MessageInput: React.FC<MessageInputProps> = ({ onSend, onPaste, disabled }) => {
  // Text area for typing or pasting
  // Paste from clipboard button
  // Send button
  // Clear button
  // Character counter
}
```

- [ ] Create component file
- [ ] Textarea with focus state
- [ ] Paste button with clipboard access
- [ ] Send button
- [ ] Clear button
- [ ] Character counter
- [ ] Style for input area

**Task 2.3: Create Suggestions component** (1.5 hours)
```typescript
// src/sidebar/components/Suggestions.tsx
interface SuggestionsProps {
  suggestions: string[];
  loading?: boolean;
  onCopy: (text: string) => void;
  onRefine: (suggestion: string) => void;
}

export const Suggestions: React.FC<SuggestionsProps> = ({ suggestions, loading, onCopy, onRefine }) => {
  // Display 3-5 AI suggestions
  // Each with [Copy] and [Refine] buttons
  // Loading state while generating
}
```

- [ ] Create component file
- [ ] Display suggestion list (3-5 items)
- [ ] Add [Copy] button per suggestion
- [ ] Add visual feedback on copy (briefly show "Copied!")
- [ ] Add [Refine] button
- [ ] Loading state (spinner while generating)
- [ ] Style for suggestions

**Task 2.4: Create Settings panel (sidebar)** (1 hour)
```typescript
// src/sidebar/components/SettingsPanel.tsx
export const SettingsPanel: React.FC = () => {
  // Quick settings in sidebar
  // Mode: [Socratic] [Direct]
  // Context: [Formal] [Friendly] [Dating]
  // LLM Provider selector
  // Link to full settings page
}
```

- [ ] Create component file
- [ ] Mode selection buttons
- [ ] Context selection buttons
- [ ] LLM provider selector
- [ ] Link to full settings
- [ ] Style for settings panel

**Task 2.5: Update Sidebar.tsx** (2 hours)
```typescript
// src/sidebar/Sidebar.tsx - MAJOR REDESIGN
export const Sidebar: React.FC = () => {
  return (
    <div className="sidebar">
      <header>Moly</header>
      
      <ChatHistory messages={messages} />
      <MessageInput onSend={handleSend} />
      <Suggestions suggestions={suggestions} />
      <SettingsPanel />
    </div>
  );
}
```

- [ ] Redesign layout for chat interface
- [ ] Import new components
- [ ] Wire up state management
- [ ] Connect to store
- [ ] Add scroll behavior
- [ ] Style complete sidebar

**Task 2.6: Update sidebar.css** (1 hour)
- [ ] Remove old detected-message styles
- [ ] Add chat history styles
- [ ] Add message input styles
- [ ] Add suggestions styles
- [ ] Add settings panel styles
- [ ] Add dark mode support
- [ ] Test responsive design

---

### Phase 3: Store Redesign - Conversation Management (4-6 hours)

**Task 3.1: Design new message/conversation types** (1 hour)
```typescript
// src/types/index.ts - UPDATE

interface Message {
  id: string;
  type: 'user' | 'moly' | 'incoming' | 'suggestion';
  content: string;
  timestamp: number;
  metadata?: {
    mode?: 'socratic' | 'direct';
    context?: 'formal' | 'friendly' | 'dating';
  };
}

interface Conversation {
  id: string;
  contactName?: string;
  contactPlatform?: string;
  messages: Message[];
  settings: {
    mode: 'socratic' | 'direct';
    context: 'formal' | 'friendly' | 'dating';
    llmProvider: string;
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
  generateSuggestions: (context: Message[]) => Promise<string[]>;
  updateSettings: (settings: Partial<Conversation['settings']>) => void;
  deleteConversation: (id: string) => void;
  exportConversation: (id: string) => string;
}
```

- [ ] Update Message interface
- [ ] Create Conversation interface
- [ ] Create ChatStore interface
- [ ] Update types/index.ts
- [ ] Verify TypeScript compiles

**Task 3.2: Redesign chatStore.ts** (1.5 hours)
```typescript
// src/stores/chatStore.ts - COMPLETE REDESIGN
export const useChatStore = create<ChatStore>((set, get) => ({
  conversations: [],
  currentConversation: null,
  
  startConversation: (contact) => set((state) => {
    const conversation: Conversation = {
      id: generateId(),
      contactName: contact,
      messages: [],
      settings: {
        mode: 'socratic',
        context: 'friendly',
        llmProvider: 'claude',
      },
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };
    return {
      conversations: [...state.conversations, conversation],
      currentConversation: conversation,
    };
  }),
  
  addMessage: (content, type) => set((state) => {
    if (!state.currentConversation) return state;
    
    const message: Message = {
      id: generateId(),
      type,
      content,
      timestamp: Date.now(),
    };
    
    return {
      currentConversation: {
        ...state.currentConversation,
        messages: [...state.currentConversation.messages, message],
        updatedAt: Date.now(),
      },
    };
  }),
  
  generateSuggestions: async (context) => {
    // Call LLM with context
    // Return suggestions
  },
  
  // ... other actions
}));
```

- [ ] Create new store structure
- [ ] Implement startConversation
- [ ] Implement loadConversation
- [ ] Implement addMessage
- [ ] Implement generateSuggestions
- [ ] Implement updateSettings
- [ ] Implement deleteConversation
- [ ] Implement exportConversation
- [ ] Test with React DevTools

**Task 3.3: Update conversationStore.ts** (1 hour)
- [ ] Rename to contactContextStore or similar
- [ ] Store per-contact context/history
- [ ] Implement load/save
- [ ] Integrate with chatStore

**Task 3.4: Minimize contactStore.ts** (1 hour)
- [ ] Keep only essential contact info
- [ ] Remove if not needed for v2.0
- [ ] Mark for cleanup in v2.1

---

### Phase 4: LLM Integration & Suggestions (4-6 hours)

**Task 4.1: Update suggestion generation** (2 hours)
- [ ] Modify LLM call to use conversation context
- [ ] Create prompt template for Socratic mode
- [ ] Create prompt template for Direct mode
- [ ] Create prompt template for each context (Formal/Friendly/Dating)
- [ ] Test with Claude provider
- [ ] Test with OpenAI provider
- [ ] Test with Ollama

**Task 4.2: Wire suggestions to UI** (1.5 hours)
- [ ] Connect store to Suggestions component
- [ ] Handle loading state
- [ ] Handle errors gracefully
- [ ] Show "Copied!" feedback on copy button
- [ ] Test all three LLM providers

**Task 4.3: Implement copy to clipboard** (1 hour)
- [ ] Use `navigator.clipboard.writeText()`
- [ ] Add error handling
- [ ] Show success feedback
- [ ] Test on different browsers

---

### Phase 5: Settings & Configuration (2-3 hours)

**Task 5.1: Update Settings.tsx** (1 hour)
- [ ] Keep LLM provider configuration
- [ ] Remove platform detection settings
- [ ] Keep chat mode selection
- [ ] Keep communication context selection
- [ ] Verify API keys work

**Task 5.2: Update Popup** (1 hour)
- [ ] Remove auto-detection logic
- [ ] Add "Open Sidebar" button
- [ ] Keep settings link
- [ ] Show LLM status
- [ ] Test popup functions

**Task 5.3: Test settings persistence** (1 hour)
- [ ] Settings save to chrome.storage
- [ ] Settings load on startup
- [ ] LLM provider switching works
- [ ] Mode/context changes work

---

### Phase 6: Testing & Polish (4-6 hours)

**Task 6.1: Unit testing** (1.5 hours)
- [ ] Test chatStore actions
- [ ] Test message addition
- [ ] Test conversation switching
- [ ] Test suggestion generation
- [ ] Test settings updates

**Task 6.2: Integration testing** (1.5 hours)
- [ ] Test sidebar with real LLM
- [ ] Test copy button workflow
- [ ] Test mode/context switching
- [ ] Test conversation persistence
- [ ] Test export functionality

**Task 6.3: Manual testing** (1 hour)
- [ ] Test on Chrome
- [ ] Test on different websites
- [ ] Test all LLM providers
- [ ] Test error scenarios
- [ ] Test dark mode

**Task 6.4: Polish & optimization** (1 hour)
- [ ] Improve animations
- [ ] Optimize performance
- [ ] Fix accessibility issues
- [ ] Clean up console warnings
- [ ] Final styling pass

---

### Phase 7: Build & Verification (2-3 hours)

**Task 7.1: Build & test** (1 hour)
- [ ] Run `npm run build`
- [ ] Verify no errors
- [ ] Check bundle size
- [ ] Load in Chrome
- [ ] Test all features

**Task 7.2: Prepare for Chrome Web Store** (1 hour)
- [ ] Create store screenshots
- [ ] Write store description
- [ ] Prepare privacy policy link
- [ ] Create promotional images

**Task 7.3: Documentation** (1 hour)
- [ ] Update inline code comments
- [ ] Verify all docs are current
- [ ] Create development guide
- [ ] Create troubleshooting guide

---

## Task Dependencies

```
Phase 1 (Cleanup)
    ↓
Phase 2 (Sidebar)
    ↓
Phase 3 (Stores) ← Can start in parallel with Phase 2
    ↓
Phase 4 (LLM Integration) ← Needs Phase 2 & 3
    ↓
Phase 5 (Settings) ← Needs Phase 3 & 4
    ↓
Phase 6 (Testing) ← Needs all above
    ↓
Phase 7 (Build & Verify) ← Final phase
```

---

## Success Criteria

- [ ] No TypeScript errors
- [ ] No console errors
- [ ] All features working:
  - [ ] Chat with Moly works
  - [ ] Mode/context selection works
  - [ ] Paste message works
  - [ ] Suggestions generate
  - [ ] Copy button works
  - [ ] Conversation history saved
  - [ ] Settings persist
- [ ] All LLM providers work
- [ ] Chrome Web Store ready

---

## Next Steps

1. **Start Phase 1**: Delete src/content/
2. **Simplify manifest**: Remove old sections
3. **Build & verify**: Test each step

Ready to start Phase 1?

---

*Implementation Plan v1.0*  
*Ready to Execute*
