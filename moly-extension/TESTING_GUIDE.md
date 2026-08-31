# Moly v2 Testing Guide

## Phase 5: Comprehensive Testing

### Prerequisites
1. Extension built: `npm run build`
2. Chrome/Chromium browser with Developer mode enabled
3. Loaded extension via `chrome://extensions/` → Load unpacked → select `dist/`

---

## Test Categories

### 1. Extension Installation & Basics
- [ ] Extension loads without errors
- [ ] Extension icon appears in toolbar
- [ ] Clicking extension icon opens sidebar
- [ ] Sidebar displays without UI errors
- [ ] Settings page accessible and displays correctly
- [ ] No console errors on startup

### 2. Settings Configuration

#### LLM Provider Setup
- [ ] Can switch between Claude, OpenAI, and Ollama tabs
- [ ] Claude tab shows API key input
- [ ] OpenAI tab shows API key input
- [ ] Ollama tab shows Base URL input
- [ ] Model dropdown appears when provider supports it
- [ ] "Get key →" links work correctly

#### Validation & Testing
- [ ] Can enter and save Claude API key
- [ ] Can enter and save OpenAI API key
- [ ] Can enter Ollama base URL
- [ ] "Save & Validate" button validates configuration
- [ ] Success message appears on valid config
- [ ] Error message appears on invalid config
- [ ] "Make Active" button sets provider as active
- [ ] Active provider status displayed correctly

#### Preferences
- [ ] Can select chat mode (Socratic/Direct)
- [ ] Can select communication context (Formal/Friendly/Dating)
- [ ] Selections persist after page reload
- [ ] Default context shows in sidebar SettingsPanel

#### Advanced Options
- [ ] "Clear All Settings" button works
- [ ] Confirmation dialog appears before clearing
- [ ] Settings are actually reset after confirmation

### 3. Sidebar - Chat Interface

#### Input Area
- [ ] Text input textarea focuses and accepts text
- [ ] Character counter works (shows current/5000)
- [ ] Can type multiple lines
- [ ] Ctrl+Enter sends message (or check for Send button)
- [ ] "Clear" button clears the input
- [ ] "Paste" button reads from clipboard
- [ ] Paste button shows "Copied!" feedback

#### Chat History
- [ ] User messages display with timestamp
- [ ] "User" type message shows with correct styling
- [ ] Incoming messages display if added
- [ ] Moly responses display if received
- [ ] Suggestion type messages display
- [ ] Delete button removes messages
- [ ] Export button downloads JSON file
- [ ] Empty state message shows when no messages

#### Suggestions Display
- [ ] Suggestions appear after sending message
- [ ] Shows 3-5 suggestions (or configurable amount)
- [ ] Each suggestion has Copy button
- [ ] Copy button copies text to clipboard
- [ ] "Copied!" feedback shows briefly
- [ ] Loading state with spinner while generating
- [ ] Error message displays on failure
- [ ] Clearing suggestions works

#### Settings Panel
- [ ] Mode selector shows current mode
- [ ] Context selector shows current context
- [ ] LLM provider displayed correctly
- [ ] Mode changes reflected in conversation
- [ ] Context changes reflected in suggestions
- [ ] Settings button opens Settings page

### 4. LLM Integration

#### Claude Provider
- [ ] Requires valid API key
- [ ] Generates suggestions within 5 seconds
- [ ] Suggestions are grammatically correct
- [ ] Suggestions match selected mode (Socratic/Direct)
- [ ] Suggestions match selected context (Formal/Friendly/Dating)
- [ ] Conversation history improves suggestions

#### OpenAI Provider
- [ ] Requires valid API key
- [ ] Can select GPT-4 or GPT-3.5 Turbo
- [ ] Generates suggestions within 3 seconds
- [ ] Suggestions quality comparable to Claude
- [ ] Works with selected modes and contexts

#### Ollama Provider
- [ ] Accepts base URL (default: http://localhost:11434)
- [ ] No API key required
- [ ] Auto-discovers available models
- [ ] Generates suggestions (speed depends on hardware)
- [ ] Works offline (no internet required)
- [ ] Error message if Ollama not running

### 5. Conversation Features

#### Message Management
- [ ] Can send multiple messages in sequence
- [ ] Conversation history persists after page reload
- [ ] Can delete individual messages
- [ ] Deletion reflected immediately
- [ ] Export includes full conversation
- [ ] Exported JSON can be imported back

#### Context Building
- [ ] Conversation history used for suggestions
- [ ] Earlier messages inform later suggestions
- [ ] Adding contact name stores properly
- [ ] Contact platform stored with conversation

#### Suggestion Selection
- [ ] Copy button works for all suggestions
- [ ] Copied text is exact match to suggestion
- [ ] Multiple suggestions can be copied sequentially
- [ ] Suggestions added to history as "suggestion" type

### 6. Data Persistence

#### Local Storage
- [ ] Conversations saved to Chrome storage
- [ ] Settings persist across browser restarts
- [ ] History available across browser sessions
- [ ] No data lost on extension update

#### Export/Import
- [ ] Export downloads valid JSON file
- [ ] Filename includes contact name and timestamp
- [ ] Can open exported file in text editor
- [ ] JSON format is readable and valid

### 7. UI/UX

#### Responsive Design
- [ ] Sidebar displays at reasonable width
- [ ] Text readable without horizontal scroll
- [ ] Buttons and inputs properly sized
- [ ] No overlapping elements

#### Visual Feedback
- [ ] Buttons show hover state
- [ ] Active selections highlighted
- [ ] Loading states show spinner
- [ ] Error states show red text
- [ ] Success states show green feedback

#### Accessibility
- [ ] Keyboard navigation works (Tab/Shift+Tab)
- [ ] Focus visible on all interactive elements
- [ ] Form labels associated with inputs
- [ ] Links have appropriate contrast

### 8. Error Handling

#### Configuration Errors
- [ ] Missing API key shows clear error
- [ ] Invalid API key shows clear error
- [ ] Network error shows user-friendly message
- [ ] Provider misconfiguration prevented

#### Runtime Errors
- [ ] No unhandled exceptions in console
- [ ] Graceful handling of LLM timeout
- [ ] Graceful handling of network failure
- [ ] User can recover from errors

### 9. Cross-Browser Testing

#### Chrome (Primary)
- [ ] All features work
- [ ] No console errors
- [ ] Extension loads correctly

#### Chromium-based (Edge, Brave, etc.)
- [ ] All features work
- [ ] Extension loads correctly
- [ ] No platform-specific issues

### 10. Performance

#### Loading
- [ ] Sidebar loads within 1 second
- [ ] Settings page loads within 1 second
- [ ] No noticeable lag during typing

#### Generation
- [ ] Suggestions generate within 5-10 seconds
- [ ] No UI freezing during generation
- [ ] Loading spinner shows throughout

#### Memory
- [ ] No memory leaks after extended use
- [ ] No slowdown after many messages
- [ ] Export doesn't crash with large histories

---

## Test Execution Checklist

### Day 1: Basic Functionality
- [ ] Install extension
- [ ] Configure Claude provider
- [ ] Send test messages
- [ ] Generate suggestions
- [ ] Copy suggestions
- [ ] Export conversation

### Day 2: All Providers
- [ ] Test OpenAI configuration
- [ ] Test OpenAI suggestions
- [ ] Test Ollama (if available)
- [ ] Compare suggestion quality

### Day 3: Settings & Preferences
- [ ] Test all chat modes
- [ ] Test all communication contexts
- [ ] Test provider switching
- [ ] Test settings persistence

### Day 4: Edge Cases
- [ ] Very long messages (4000+ chars)
- [ ] Special characters (emoji, accents, etc.)
- [ ] Multiple rapid suggestion requests
- [ ] Network interruptions
- [ ] Invalid provider credentials

### Day 5: Performance & Polish
- [ ] Extended testing (50+ messages)
- [ ] Browser restart with data persistence
- [ ] Export/Import cycle
- [ ] Visual consistency across UI
- [ ] Cross-browser verification

---

## Known Limitations & Notes

1. **Offline Mode**: Only Ollama works without internet
2. **Rate Limiting**: LLM providers may rate limit rapid requests
3. **Storage**: Chrome storage limit is 10MB (sufficient for this use case)
4. **Performance**: Suggestion generation depends on LLM provider speed
5. **Models**: Available models depend on provider and account tier

---

## Bug Report Template

```
### Issue: [Brief Title]

**Environment**:
- Browser: [Chrome/Edge/Brave]
- OS: [Windows/macOS/Linux]
- Extension Version: [Version from About]

**Steps to Reproduce**:
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Expected Behavior**:
[What should happen]

**Actual Behavior**:
[What actually happens]

**Screenshots/Console Logs**:
[Attach if applicable]

**Workaround** (if any):
[Temporary solution if exists]
```

---

## Testing Sign-Off

- [ ] All basic functionality tests pass
- [ ] All provider tests pass
- [ ] All edge case tests pass
- [ ] No critical bugs found
- [ ] Performance acceptable
- [ ] Ready for Chrome Web Store submission

---

*Last Updated: 2026-09-01*
*Tester: [Your Name]*
