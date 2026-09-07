# Moly Frontend Architecture - Complete Specification

**Date**: September 6, 2026  
**Status**: Design Specification  
**Version**: 1.0

---

## Overview

The extension front-end is the user's window into Moly. It must be:
- **Lightweight** — Runs in browser, fast response
- **Stateful** — Manages conversations, contacts, settings
- **Responsive** — Works on any window size
- **Offline-capable** — Functions even when backend down
- **Unobtrusive** — Doesn't interfere with user's browsing

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────┐
│ Chrome Extension (Manifest V3)                      │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ Service Worker (background.js)               │  │
│  │ - Lifecycle management                       │  │
│  │ - Alarms and timers                          │  │
│  │ - Permission requests                        │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ Content Scripts                              │  │
│  │ - None (user-driven, not auto-reading)       │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ Side Panel (React SPA)                       │  │
│  │                                              │  │
│  │  ┌─ Sidebar.tsx (Root Component)             │  │
│  │  │  ├─ Header (Moly logo, settings button)  │  │
│  │  │  ├─ Conversation Selector                │  │
│  │  │  ├─ Chat Area                            │  │
│  │  │  │  ├─ Messages                          │  │
│  │  │  │  ├─ Spinner (during processing)       │  │
│  │  │  │  └─ Error messages                    │  │
│  │  │  ├─ Controls (Mode, Tone, Context)       │  │
│  │  │  ├─ Suggestions (scrollable)             │  │
│  │  │  ├─ Chat Input                           │  │
│  │  │  └─ ReflectionModal (overlay)            │  │
│  │  │                                          │  │
│  │  └─ Settings.tsx (Modal/Page)               │  │
│  │     ├─ Provider selection (Claude/OpenAI)  │  │
│  │     ├─ API key input                        │  │
│  │     ├─ Model selection                      │  │
│  │     ├─ Context mode                         │  │
│  │     └─ About Me editor                      │  │
│  │                                              │  │
│  │  └─ NewConversationModal.tsx (overlay)      │  │
│  │     ├─ Contact selector (searchable)        │  │
│  │     ├─ Create new contact button            │  │
│  │     ├─ Relationship type selector           │  │
│  │     └─ Start button                         │  │
│  │                                              │  │
│  │  └─ ReflectionModal.tsx (overlay)           │  │
│  │     ├─ Extracted characteristics            │  │
│  │     ├─ Extracted interests                  │  │
│  │     ├─ Extracted communication prefs        │  │
│  │     ├─ Edit controls                        │  │
│  │     └─ Approve/Skip buttons                 │  │
│  │                                              │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ Zustand Stores                               │  │
│  │                                              │  │
│  │  ├─ chatStore                               │  │
│  │  │  ├─ conversations: {}                    │  │
│  │  │  ├─ currentConversation: { ... }         │  │
│  │  │  ├─ messages: [ ... ]                    │  │
│  │  │  ├─ isLoading: boolean                   │  │
│  │  │  └─ methods: addMessage(), etc.          │  │
│  │  │                                          │  │
│  │  ├─ settingsStore                          │  │
│  │  │  ├─ provider: string                     │  │
│  │  │  ├─ apiKey: string (masked)              │  │
│  │  │  ├─ model: string                        │  │
│  │  │  ├─ mode: "socratic" | "direct"          │  │
│  │  │  ├─ tone: "formal" | "friendly"          │  │
│  │  │  ├─ context: { ... }                     │  │
│  │  │  └─ methods: updateSettings(), etc.      │  │
│  │  │                                          │  │
│  │  └─ contactStore                           │  │
│  │     ├─ contacts: {}                        │  │
│  │     ├─ aboutMe: { ... }                    │  │
│  │     └─ methods: addContact(), etc.         │  │
│  │                                              │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ API Clients (src/api/)                       │  │
│  │                                              │  │
│  │  ├─ backendManager.ts                       │  │
│  │  │  └─ Handles backend URL detection        │  │
│  │  │                                          │  │
│  │  ├─ conversationAPI.ts                      │  │
│  │  │  └─ POST /api/conversation/generate      │  │
│  │  │                                          │  │
│  │  ├─ providerManager.ts                      │  │
│  │  │  ├─ Claude provider (API key auth)       │  │
│  │  │  ├─ OpenAI provider                      │  │
│  │  │  └─ Ollama provider (local)              │  │
│  │  │                                          │  │
│  │  └─ molyAgent.ts                            │  │
│  │     └─ Calls backend agent endpoints        │  │
│  │                                              │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ Storage Layer (chrome.storage)               │  │
│  │                                              │  │
│  │  ├─ chrome.storage.local (persistent)       │  │
│  │  │  ├─ conversations/                       │  │
│  │  │  ├─ contacts/                            │  │
│  │  │  ├─ aboutMe/                             │  │
│  │  │  └─ settings/                            │  │
│  │  │                                          │  │
│  │  └─ chrome.storage.session (temp)           │  │
│  │     └─ UI state (modals open, etc.)         │  │
│  │                                              │  │
│  └──────────────────────────────────────────────┘  │
│                                                     │
└─────────────────────────────────────────────────────┘
        ↓ HTTP/JSON
┌─────────────────────────────────────────────────────┐
│ Backend (Go + Claude API)                           │
└─────────────────────────────────────────────────────┘
```

---

## Component Hierarchy

### Root: Sidebar.tsx

```typescript
// Main container component
interface SidebarProps {}

interface SidebarState {
  // Conversations
  conversations: Conversation[];
  currentConversation: Conversation | null;
  conversationMembers: Contact[]; // Fresh contact data
  messages: Message[];
  
  // UI State
  isLoading: boolean;
  processingStage: string; // "Analyzing...", "Generated X suggestions in Xs"
  processingSeconds: number;
  showSettings: boolean;
  showNewConversation: boolean;
  showReflection: boolean;
  selectedReflection: Reflection | null;
  error: string | null;
  
  // Controls
  mode: 'socratic' | 'direct';
  tone: 'formal' | 'friendly' | 'dating';
  context: string;
  
  // Suggestions
  suggestions: Suggestion[];
  selectedSuggestionIndex: number | null;
}

export const Sidebar: React.FC<SidebarProps> = () => {
  // Hooks:
  // - useSettingsStore()
  // - useChatStore()
  // - useContactStore()
  // - useState() for local UI state
  // - useEffect() for data loading + lifecycle
  // - useCallback() for event handlers
  
  return (
    <div className="sidebar-container">
      <SidebarHeader />
      <div className="sidebar-content">
        <ConversationSelector />
        <ControlsSection />
        <ChatSection />
        <SuggestionsSection />
        <ChatInputForm />
      </div>
      {showSettings && <Settings />}
      {showNewConversation && <NewConversationModal />}
      {showReflection && <ReflectionModal />}
      {error && <ErrorBanner />}
    </div>
  );
};
```

### Settings.tsx

```typescript
interface SettingsState {
  selectedProvider: 'claude' | 'openai' | 'ollama';
  apiKey: string; // Masked if saved
  selectedModel: string;
  discoveredModels: string[];
  isDiscovering: boolean;
  discoveryStatus: string;
  error: string | null;
}

export const Settings: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  // Flow:
  // 1. User selects provider
  // 2. User enters API key
  // 3. User clicks "Discover Models"
  // 4. Backend auto-detects available models
  // 5. User selects model
  // 6. User clicks "Save"
  // 7. Settings stored in chrome.storage
  
  // Key implementation:
  // - Arrow functions defined BEFORE useEffect (JavaScript hoisting)
  // - Create fresh provider instances (bypass validation)
  // - When masked key: use stored key from settings
  
  return (
    <div className="settings-modal">
      <h2>Settings</h2>
      <ProviderSelector />
      <APIKeyInput />
      <DiscoverModelsButton />
      <ModelSelector />
      <SaveButton />
      <AboutMeEditor />
    </div>
  );
};
```

### NewConversationModal.tsx

```typescript
interface NewConversationModalState {
  selectedContacts: string[]; // contact IDs
  selectedRelationshipType: string;
  newContactName: string;
  isCreating: boolean;
  error: string | null;
}

export const NewConversationModal: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  // Flow:
  // 1. Load existing contacts (with checkboxes)
  // 2. Allow multiple contact selection
  // 3. Allow creating new contact inline
  // 4. Click "Start" creates conversation
  
  // Key implementation:
  // - Checkbox onChange: direct call to handleToggleContact()
  // - NO preventDefault() or stopPropagation()
  // - Contact selection works smoothly
  
  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h2>New Conversation</h2>
        <ContactList />
        <CreateNewContactForm />
        <StartButton />
      </div>
    </div>
  );
};
```

### ReflectionModal.tsx

```typescript
interface ReflectionModalProps {
  reflection: Reflection;
  onApprove: (edits: Partial<Reflection>) => void;
  onSkip: () => void;
}

export const ReflectionModal: React.FC<ReflectionModalProps> = ({
  reflection,
  onApprove,
  onSkip
}) => {
  // Shows extracted insights for user verification
  // User can:
  // - Approve as-is
  // - Edit characteristics, interests, preferences
  // - Add notes
  // - Skip for now
  
  return (
    <div className="modal-overlay">
      <div className="modal-content">
        <h3>Learning from this conversation...</h3>
        <CharacteristicsEditor />
        <InterestsEditor />
        <CommunicationPreferencesEditor />
        <ApproveButton />
        <SkipButton />
      </div>
    </div>
  );
};
```

### Suggestions Section

```typescript
interface SuggestionsProps {
  suggestions: Suggestion[];
  processingStage: string;
  processingSeconds: number;
  isLoading: boolean;
}

export const Suggestions: React.FC<SuggestionsProps> = ({
  suggestions,
  processingStage,
  processingSeconds,
  isLoading
}) => {
  return (
    <div className="suggestions-section">
      {isLoading ? (
        <div className="loading">
          <div className="spinner" />
          <span>
            {processingStage}
            {processingSeconds > 2 && ` (${processingSeconds}s)`}
            {processingSeconds > 3 && ` (models can be slow on older systems)`}
          </span>
        </div>
      ) : suggestions.length > 0 ? (
        <div className="suggestions-list">
          {suggestions.map((suggestion, index) => (
            <SuggestionCard
              key={index}
              suggestion={suggestion}
              index={index}
              onCopy={handleCopySuggestion}
            />
          ))}
        </div>
      ) : (
        <div className="empty-state">No suggestions yet. Start a conversation!</div>
      )}
    </div>
  );
};
```

---

## State Management (Zustand Stores)

### chatStore.ts

```typescript
interface Conversation {
  id: string;
  userId: string;
  contactId: string;
  contactName: string;
  messages: Message[];
  createdAt: number;
  updatedAt: number;
}

interface Message {
  id: string;
  conversationId: string;
  role: 'user' | 'assistant';
  content: string;
  type: 'message' | 'question' | 'suggestion';
  metadata?: {
    tone?: string;
    mode?: string;
    chosenIndex?: number;
  };
  timestamp: number;
}

interface ChatStore {
  // State
  conversations: Record<string, Conversation>;
  currentConversationId: string | null;
  messages: Message[];
  isLoading: boolean;
  processingStage: string;
  processingSeconds: number;
  error: string | null;

  // Actions
  loadConversations(): Promise<void>;
  selectConversation(conversationId: string): Promise<void>;
  createConversation(contactId: string, contactName: string): Promise<void>;
  addMessage(message: Message): Promise<void>;
  setLoading(loading: boolean): void;
  setProcessingStage(stage: string): void;
  setError(error: string | null): void;
  clear(): void;
}

export const useChatStore = create<ChatStore>((set, get) => ({
  // Implementation
}));
```

### settingsStore.ts

```typescript
interface SettingsStore {
  // State
  provider: 'claude' | 'openai' | 'ollama';
  apiKey: string; // Stored encrypted in chrome.storage
  model: string;
  mode: 'socratic' | 'direct';
  tone: 'formal' | 'friendly' | 'dating';
  context: string;

  // Actions
  loadSettings(): Promise<void>;
  updateProvider(provider: string): Promise<void>;
  updateApiKey(key: string): Promise<void>;
  updateModel(model: string): Promise<void>;
  updateMode(mode: string): Promise<void>;
  updateTone(tone: string): Promise<void>;
  updateContext(context: string): Promise<void>;
  saveSettings(): Promise<void>;
  clear(): void;
}

export const useSettingsStore = create<SettingsStore>((set, get) => ({
  // Implementation
}));
```

### contactStore.ts

```typescript
interface Contact {
  id: string;
  userId: string;
  name: string;
  relationship: 'close_friend' | 'family' | 'work' | 'romantic' | 'new';
  characteristics: string[];
  interests: string[];
  communicationPreferences: string;
  notes: string;
  reflections: Reflection[];
  createdAt: number;
  updatedAt: number;
}

interface AboutMe {
  userId: string;
  communicationStyle: string;
  values: string[];
  preferredTone: string;
  notes: string;
  updatedAt: number;
}

interface ContactStore {
  // State
  contacts: Record<string, Contact>;
  aboutMe: AboutMe | null;

  // Actions
  loadContacts(): Promise<void>;
  loadAboutMe(): Promise<void>;
  addContact(contact: Contact): Promise<void>;
  updateContact(contactId: string, updates: Partial<Contact>): Promise<void>;
  deleteContact(contactId: string): Promise<void>;
  updateAboutMe(updates: Partial<AboutMe>): Promise<void>;
  mergeReflection(contactId: string, reflection: Reflection): Promise<void>;
  clear(): void;
}

export const useContactStore = create<ContactStore>((set, get) => ({
  // Implementation
}));
```

---

## API Clients

### conversationAPI.ts

```typescript
export interface ConversationRequest {
  conversationId: string;
  userId: string;
  userMessage: string;
  mode: 'socratic' | 'direct';
  tone: 'formal' | 'friendly' | 'dating';
}

export interface ConversationResponse {
  phase: 'suggestions_ready' | 'context_gathering' | 'intention_gathering' | 'safety_alert' | 'error';
  suggestions: Suggestion[];
  questions: string[];
  reflection: Reflection | null;
  riskWarning: RiskWarning | null;
  safetyAlert: SafetyAlert | null;
  processingTimeMs: number;
}

export async function generateConversationResponse(
  request: ConversationRequest
): Promise<ConversationResponse> {
  const backendUrl = getBackendManager().getBackendUrl();
  
  const response = await fetch(`${backendUrl}/api/conversation/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request)
  });
  
  if (!response.ok) {
    // Handle errors
  }
  
  return response.json();
}
```

### molyAgent.ts

```typescript
export async function checkSafety(message: string): Promise<SafetyCheckResult> {
  // Calls backend safety checker
}

export async function generateQuestions(
  contactName: string,
  context: string
): Promise<QuestionGeneratorResult> {
  // Calls backend question generator
}

export async function generateSuggestions(
  aboutMe: AboutMe,
  contactProfile: Contact,
  intention: string,
  history: Message[],
  mode: string,
  tone: string,
  userMessage: string
): Promise<Suggestion[]> {
  // Calls backend suggestion generator
}
```

---

## Data Flow Example: User Sends Message

```
1. USER INTERACTION:
   └─ Types message in chat input
   └─ Presses Send button
   
2. EVENT HANDLER (onSendMessage):
   ├─ Get message from input
   ├─ Add to local messages array (optimistic)
   ├─ Update Zustand store
   ├─ Show loading spinner
   ├─ Clear input
   └─ Prepare backend request

3. PREPARE REQUEST:
   ├─ Get conversationId from current state
   ├─ Get userId from settings
   ├─ Get mode/tone from settings
   ├─ Get message content
   └─ Build ConversationRequest

4. CALL BACKEND:
   ├─ POST /api/conversation/generate
   ├─ Wait for response (1-3 seconds)
   ├─ Show processing spinner with elapsed time
   └─ After 2 seconds show "models can be slow..." message

5. HANDLE RESPONSE:
   ├─ If error:
   │  ├─ Show error banner
   │  ├─ Keep local message (not lost)
   │  └─ Let user retry
   ├─ If safety alert:
   │  ├─ Show alert with resources
   │  ├─ Don't show suggestions
   │  └─ User can continue or abandon
   ├─ If risk warning:
   │  ├─ Show warning + educational questions
   │  ├─ Show suggestions anyway
   │  └─ User can proceed or reconsider
   └─ If success:
      ├─ Add Moly's response to messages
      ├─ Display suggestions (3 cards)
      ├─ Show reflection modal
      ├─ Hide loading spinner
      └─ Clear any errors

6. USER ACTS:
   ├─ Copy suggestion → To clipboard
   ├─ Request modification → Send new request
   ├─ Approve reflection → Save to contact profile
   └─ Continue conversation → Type new message

7. STORE UPDATES:
   ├─ chatStore: Add messages
   ├─ contactStore: Merge reflection
   ├─ Backend learning: Record all interactions
   └─ Next conversation: Richer context available
```

---

## Offline Capability

**When Backend Down:**
```
1. Extension detects backend unavailable
   ├─ 504 error
   ├─ Timeout
   └─ Network error

2. Graceful degradation:
   ├─ Load cached contacts/conversations
   ├─ Show "Backend unavailable" banner
   ├─ Offer fallback:
   │  ├─ "Continue in offline mode (limited features)"
   │  ├─ "Or check your backend setup"
   │  └─ "Retry" button
   
3. Offline mode:
   ├─ Load cached suggestions from last conversation
   ├─ Show simple fallback questions
   ├─ Don't block chat input
   ├─ Store messages locally
   └─ Sync when backend returns

4. When backend recovers:
   ├─ Automatic retry
   ├─ Sync stored messages
   ├─ Resume normal operation
```

---

## Error Handling

| Error | User Sees | Action |
|-------|-----------|--------|
| Backend unavailable | "Backend is offline. Try again?" | Retry button |
| Invalid API key | "API key invalid. Check settings." | Go to Settings |
| Model not found | "Model not available. Discover models?" | Auto-discover |
| Rate limited | "Too many requests. Wait a moment..." | Retry after delay |
| Malformed response | "Unexpected response. Try again?" | Retry |
| Network timeout | "Request timed out. Try again?" | Retry |
| Chrome storage quota | "Storage full. Clear old conversations?" | Cleanup UI |

---

## Performance Considerations

**Bundle Size:**
- React: ~50KB
- Zustand: ~5KB
- Components: ~30KB
- Total: ~100KB (gzipped)

**Memory Usage:**
- Conversations in memory: Last 5 loaded
- Messages per conversation: Last 50 loaded
- Contacts: All loaded (typically < 50)
- Total: ~5-10MB typical

**Optimization Strategies:**
- Lazy load old conversations
- Virtualize long message lists
- Debounce store updates
- Cache API responses
- Optimize re-renders with useCallback/useMemo

---

## Browser Compatibility

- Chrome: 114+ (sidePanel API)
- Edge: 114+ (Chromium-based)
- Brave: 1.58+ (Chromium-based)
- Opera: 100+ (Chromium-based)

---

## Manifest V3 Specifics

```json
{
  "manifest_version": 3,
  "name": "Moly",
  "version": "1.0.0",
  "permissions": [
    "storage",
    "sidePanel"
  ],
  "host_permissions": [
    "<all_urls>"
  ],
  "background": {
    "service_worker": "background.js"
  },
  "side_panel": {
    "default_path": "src/sidebar/sidebar.html"
  },
  "action": {
    "default_title": "Open Moly"
  }
}
```

---

## Testing Strategy for Frontend

**Unit Tests:**
- Component rendering
- Store actions
- API client error handling
- Utility functions

**Integration Tests:**
- Full message flow
- Settings save/load
- Conversation creation
- Reflection modal workflow

**E2E Tests:**
- Open extension
- Create conversation
- Send message
- Approve reflection
- Verify data saved

---

This architecture enables a responsive, user-friendly experience while maintaining the core principle: Moly is always responsive, never blocking, and gracefully handles failures.
