# MOLY MESSAGE DETECTION MECHANISM
## How Moly Automatically Detects and Reads Messages from ANY Website

**Status:** Technical Specification  
**Version:** 1.0  
**Date:** August 4, 2026

---

## 1. OVERVIEW

### Core Principle
Moly does NOT require users to manually copy-paste messages. Instead, Moly **automatically detects when a new message arrives** on any website and **reads the message text directly from the page**.

### What This Means
```
OLD WAY (Manual):
User receives message → User copies message → User pastes to Moly

NEW WAY (Automatic):
User receives message → Moly detects automatically → Message appears in Moly chat
```

---

## 2. HOW MESSAGE DETECTION WORKS

### 2.1 Content Script Monitoring

**Moly's content script runs on every website and:**

1. **Watches the DOM (Document Object Model)**
   - Monitors page structure for changes
   - Detects when new message elements appear
   - Listens for new message notifications

2. **Extracts Message Text**
   - Reads text content from message elements
   - Identifies sender information
   - Captures message timestamp

3. **Sends to Sidebar**
   - Passes message to Moly sidebar automatically
   - Message appears pre-filled in chat interface
   - Shows notification to user

### 2.2 Technical Implementation

```
CONTENT SCRIPT WORKFLOW:

1. INITIALIZATION
   ├─ Content script injects into webpage
   ├─ Identifies website platform (Tinder, Facebook, etc.)
   ├─ Sets up DOM monitoring
   └─ Waits for message activity

2. MESSAGE DETECTION (MutationObserver)
   ├─ Watches for DOM changes
   ├─ Detects new message containers
   ├─ Identifies incoming vs outgoing messages
   ├─ Filters for messages from others (not user's own)
   └─ Triggers when new message detected

3. TEXT EXTRACTION
   ├─ Finds message text element
   ├─ Reads inner text content
   ├─ Cleans up formatting
   ├─ Extracts sender name/profile
   ├─ Captures timestamp
   └─ Builds message object

4. NOTIFICATION & DISPLAY
   ├─ Sends message to background script
   ├─ Background script notifies sidebar
   ├─ Sidebar pre-fills message in chat
   ├─ Shows notification badge
   ├─ Auto-opens sidebar (optional)
   └─ User sees message ready to reply to

5. MESSAGE OBJECT CREATED
   {
     "sender": "Sarah",
     "text": "That sounds awesome! I love hiking too.",
     "timestamp": 1691587200,
     "platform": "tinder",
     "url": "https://tinder.com/...",
     "profileId": "abc123"
   }
```

---

## 3. PLATFORM-SPECIFIC DETECTION

### 3.1 How Moly Works on Different Websites

**The beauty of Moly's design:**
- Doesn't need special code for each platform
- Uses generic DOM selectors that work universally
- Falls back to text content extraction if selectors fail

### 3.2 Detection on Major Platforms

```
TINDER.COM
├─ Detects: Message appears in chat window
├─ Selector: .Msg__content or similar
├─ Extracts: "That sounds awesome!"
├─ Context: DATING (auto-selected)
└─ Notification: "New message from Sarah"

FACEBOOK.COM
├─ Detects: Message appears in messenger
├─ Selector: .msg-content or similar
├─ Extracts: "Hey, how are you?"
├─ Context: FRIENDLY (auto-selected)
└─ Notification: "New message from Sarah"

LINKEDIN.COM
├─ Detects: Message appears in conversation
├─ Selector: Message element in DOM
├─ Extracts: "Interested in that project"
├─ Context: FORMAL (auto-selected)
└─ Notification: "New message from Sarah"

FETLIFE.COM
├─ Detects: Message in private messages
├─ Selector: Message container element
├─ Extracts: Message text
├─ Context: DATING/FRIENDLY (asks user)
└─ Notification: "New message from Sarah"

DISCORD.COM
├─ Detects: Message in DM channel
├─ Selector: Message content element
├─ Extracts: Message text
├─ Context: FRIENDLY (auto-selected)
└─ Notification: "New message from Sarah"

SLACK.COM
├─ Detects: Direct message
├─ Selector: Message text element
├─ Extracts: Message content
├─ Context: FORMAL/FRIENDLY (context-dependent)
└─ Notification: "New message from Sarah"

WHATSAPP WEB
├─ Detects: Message in chat
├─ Selector: Message bubble element
├─ Extracts: Message text
├─ Context: FRIENDLY (auto-selected)
└─ Notification: "New message from Sarah"

ANY UNKNOWN WEBSITE
├─ Detects: Text in message-like containers
├─ Selector: Generic message selectors
├─ Extracts: Visible text content
├─ Context: ASKS USER "What context?"
└─ Notification: "New message detected"
```

---

## 4. TECHNICAL DEEP DIVE

### 4.1 MutationObserver Implementation

```javascript
// Simplified example of how detection works

const observer = new MutationObserver((mutations) => {
  mutations.forEach((mutation) => {
    // Check if new nodes were added
    if (mutation.addedNodes.length > 0) {
      // Look for message-like elements
      const messageElements = Array.from(mutation.addedNodes).filter(node => 
        node.textContent && 
        (node.classList.contains('message') || 
         node.classList.contains('msg') ||
         node.getAttribute('role') === 'article')
      );

      // Extract and process message
      messageElements.forEach(element => {
        const messageText = element.textContent.trim();
        const sender = extractSenderName(element);
        
        // Send to Moly sidebar
        chrome.runtime.sendMessage({
          type: 'NEW_MESSAGE_DETECTED',
          message: {
            sender: sender,
            text: messageText,
            platform: detectPlatform(),
            timestamp: Date.now()
          }
        });
      });
    }
  });
});

// Start observing the page for changes
observer.observe(document.body, {
  childList: true,      // Watch for added/removed nodes
  subtree: true,        // Watch all descendants
  characterData: true   // Watch text content changes
});
```

### 4.2 Platform Detection Algorithm

```javascript
function detectPlatform() {
  const url = window.location.href;
  const domain = new URL(url).hostname;

  // Check domain
  if (domain.includes('tinder.com')) return 'tinder';
  if (domain.includes('bumble.com')) return 'bumble';
  if (domain.includes('hinge.com')) return 'hinge';
  if (domain.includes('match.com')) return 'match';
  if (domain.includes('facebook.com')) return 'facebook';
  if (domain.includes('linkedin.com')) return 'linkedin';
  if (domain.includes('discord.com')) return 'discord';
  if (domain.includes('slack.com')) return 'slack';
  if (domain.includes('fetlife.com')) return 'fetlife';
  if (domain.includes('web.whatsapp.com')) return 'whatsapp';
  
  // Unknown platform - use generic detection
  return 'generic';
}

function getContextForPlatform(platform) {
  const contextMap = {
    'tinder': 'dating',
    'bumble': 'dating', // or 'friendly' for BFF
    'hinge': 'dating',
    'match': 'dating',
    'facebook': 'friendly',
    'linkedin': 'formal',
    'discord': 'friendly',
    'slack': 'formal', // context-dependent
    'fetlife': 'dating', // or ask user
    'whatsapp': 'friendly',
    'generic': null // ask user
  };
  
  return contextMap[platform];
}
```

### 4.3 Message Extraction

```javascript
function extractMessageText(messageElement) {
  // Try multiple selector strategies
  
  // Strategy 1: Look for common message class patterns
  let textContent = messageElement.querySelector(
    '.message-text, .msg-content, [role="article"], .message-body'
  )?.textContent;
  
  if (textContent) return textContent.trim();
  
  // Strategy 2: Use all text content but filter out UI elements
  textContent = messageElement.innerText || messageElement.textContent;
  
  // Remove timestamps, delivery indicators, etc.
  textContent = textContent
    .replace(/(\d{1,2}:\d{2}\s*(AM|PM|am|pm))?/, '') // Remove time
    .replace(/Delivered|Seen|Read/, '') // Remove status indicators
    .trim();
  
  return textContent;
}

function extractSenderInfo(messageElement) {
  // Try to find sender name/profile
  const senderName = messageElement.querySelector(
    '.sender-name, .message-sender, [data-sender]'
  )?.textContent || 'Unknown';
  
  const senderProfile = messageElement.getAttribute('data-sender-id') || 
                       messageElement.getAttribute('data-user-id');
  
  return {
    name: senderName.trim(),
    id: senderProfile
  };
}
```

---

## 5. NOTIFICATION & USER FLOW

### 5.1 When Message Detected

```
AUTOMATIC FLOW:

1. New message arrives on website
   ↓
2. Moly content script detects it
   ↓
3. Extracts message text & sender
   ↓
4. Sends to background script
   ↓
5. Background script notifies sidebar
   ↓
6. Sidebar updates with new message
   ↓
7. Extension icon badge shows "1" (new message)
   ↓
8. Desktop notification appears: "New message from Sarah"
   ↓
9. Sidebar can auto-open (user preference)
   ↓
10. Message pre-filled in chat interface
    "Sarah: That sounds awesome! I love hiking too."
    ↓
11. User opens Moly chat
    ↓
12. Moly shows: "Context: DATING (detected)"
    ↓
13. User can accept context or override
    ↓
14. User continues chat with Moly
```

### 5.2 User Sees in Sidebar

```
┌─────────────────────────────────┐
│ 🧠 MOLY CHAT                    │
├─────────────────────────────────┤
│                                 │
│ 💬 New message from Sarah       │
│                                 │
│ Platform: Tinder (detected)     │
│ Context: DATING (auto-selected) │
│                                 │
│ ─────────────────────────────   │
│                                 │
│ Message:                        │
│ "That sounds awesome!           │
│  I love hiking too. Have you    │
│  tried [trail name]?"           │
│                                 │
│ ─────────────────────────────   │
│                                 │
│ 💡 Ready to respond?            │
│                                 │
│ [Get suggestions]               │
│ [Type manually]                 │
│ [Change context]                │
│                                 │
└─────────────────────────────────┘
```

---

## 6. HANDLING DIFFERENT MESSAGE TYPES

### 6.1 Types of Messages Detected

```
INCOMING MESSAGES (Detected ✅)
├─ Text messages from other users
├─ First messages/new conversations
├─ Replies to previous messages
├─ Group messages (with sender identified)
└─ Messages with emojis and formatting

MESSAGES IGNORED (Not detected ❌)
├─ User's own outgoing messages
├─ System messages ("User went online")
├─ Typing indicators ("User is typing...")
├─ Delivery receipts ("Delivered", "Seen")
├─ Old messages (only new arrivals)
└─ Messages user didn't receive (archived, etc.)
```

### 6.2 Filtering Logic

```javascript
function isIncomingMessage(messageElement) {
  // Check if message is from another user, not the current user
  
  // Method 1: Check sender vs current user
  const sender = extractSenderInfo(messageElement);
  const currentUser = getCurrentUserProfile();
  
  if (sender.id && currentUser.id && sender.id === currentUser.id) {
    return false; // User's own message
  }
  
  // Method 2: Check CSS classes
  if (messageElement.classList.contains('outgoing') ||
      messageElement.classList.contains('sent')) {
    return false; // User's own message
  }
  
  // Method 3: Check message bubble position/alignment
  const alignment = getComputedStyle(messageElement).textAlign;
  if (alignment === 'right') {
    return false; // Typically user's messages align right
  }
  
  // Method 4: Check if message is in user's conversation thread
  const isInThread = isMessageInUserThread(messageElement);
  
  return isInThread;
}

function isNewMessage(messageElement) {
  // Only detect messages that just arrived
  
  const messageTime = extractTimestamp(messageElement);
  const currentTime = Date.now();
  const timeDifference = currentTime - messageTime;
  
  // Only consider messages less than 5 seconds old as "new"
  return timeDifference < 5000;
}
```

---

## 7. HANDLING EDGE CASES

### 7.1 Multiple Simultaneous Messages

```
If user receives multiple messages at once:

Message 1 arrives
├─ Detected immediately
├─ Notification shown
└─ Added to queue

Message 2 arrives
├─ Detected
├─ Notification updated
└─ Queue updated

User opens Moly
├─ Shows most recent message
├─ Shows count: "3 new messages"
├─ Can scroll through message history
└─ Responds to latest
```

### 7.2 Message Updates/Edits

```
If user's message gets edited or deleted:

Original message detected
├─ Shown to user
└─ Stored in memory

Message gets edited
├─ Content script detects DOM change
├─ Updates message in Moly
├─ Shows: "Message edited"
└─ User can see updated version

Message gets deleted
├─ Content script detects removal
├─ Removes from Moly
├─ Shows: "Message deleted"
└─ User can see conversation history
```

### 7.3 Long Messages / Collapsed Text

```
If message is very long:

Moly detects full message text
├─ Extracts entire content
├─ Shows in chat interface
├─ Highlights key parts
└─ Can expand/collapse

If website shows "[...] Read more":

Moly clicks to expand (optional)
├─ Waits for content to load
├─ Extracts full expanded text
├─ Shows complete message
└─ Generates suggestions based on full context
```

---

## 8. PERMISSIONS REQUIRED

### 8.1 Manifest Permissions

```json
{
  "permissions": [
    "activeTab",
    "scripting",
    "storage",
    "notifications",
    "webRequest"
  ],
  "host_permissions": [
    "<all_urls>"
  ],
  "content_scripts": [
    {
      "matches": ["<all_urls>"],
      "js": ["content-script.js"],
      "run_at": "document_end"
    }
  ]
}
```

### 8.2 Why Each Permission

```
activeTab
├─ Access current tab where message is
└─ Needed to detect messages

scripting
├─ Inject content script into webpages
└─ Needed to run detection code

storage
├─ Store messages, contacts, settings locally
└─ Needed for IndexedDB access

notifications
├─ Show desktop notifications
└─ Needed to alert user of new messages

host_permissions (<all_urls>)
├─ Run on any website
└─ Needed for universal message detection
```

---

## 9. PERFORMANCE CONSIDERATIONS

### 9.1 Optimization Strategies

```
EFFICIENT DETECTION:

1. Debounce Detection
   ├─ Don't check every DOM change
   ├─ Wait 100-200ms between checks
   ├─ Prevents excessive processing
   └─ Reduces CPU usage

2. Targeted Selectors
   ├─ Use specific selectors first
   ├─ Fall back to generic only if needed
   ├─ Faster searches in large DOMs
   └─ Less computation

3. Lazy Loading
   ├─ Only process visible messages
   ├─ Don't process archived conversations
   ├─ Stop processing when sidebar closed
   └─ Saves memory

4. Message Caching
   ├─ Remember last detected message ID
   ├─ Skip re-processing same message
   ├─ Prevents duplicates
   └─ Improves speed
```

### 9.2 Resource Usage

```
CPU Impact:
├─ Idle: < 0.5% (just listening)
├─ Message arrives: 1-2% (brief spike)
├─ Processing: < 1% (then returns to idle)
└─ Total: Negligible

Memory Impact:
├─ Content script: ~2-3 MB
├─ Message queue: ~500KB (typical)
├─ Total: Very small

Network:
├─ No network traffic for detection
├─ Only API call when user requests suggestions
└─ Zero overhead
```

---

## 10. FALLBACK STRATEGIES

### 10.1 If Automatic Detection Fails

```
SCENARIO 1: Website structure unknown

Fallback Option 1: Generic text detection
├─ Look for any new text in message-like areas
├─ Extract by position on page
├─ Less accurate but functional
└─ User can confirm/edit

Fallback Option 2: Manual selection
├─ Show tooltip: "Select message text"
├─ User highlights message
├─ Moly extracts selected text
└─ Better accuracy

Fallback Option 3: Copy-paste option
├─ If all else fails, user can copy
├─ Moly provides paste interface
├─ Manual but reliable
└─ Last resort
```

### 10.2 If Platform Not Recognized

```
SCENARIO 2: Unknown website

Automatic handling:
├─ Detect message as generic
├─ Ask user: "What context?"
├─ User selects: Formal/Friendly/Dating
├─ Moly proceeds as normal
└─ Works on any website
```

---

## 11. PRIVACY & SECURITY

### 11.1 What Moly Reads

```
READS FROM PAGE:
✓ Message text content (what user needs to reply to)
✓ Sender name/profile info (who is messaging)
✓ Message timestamp (when message arrived)

DOES NOT READ:
✗ User's account passwords
✗ User's personal account info (beyond sender name)
✗ Website tracking data
✗ Cookies or authentication tokens
✗ Any data user didn't receive
✗ Other users' private data
```

### 11.2 Where Data Goes

```
Message Detection Flow:

Message detected on page
    ↓
Stored in content script memory
    ↓
Passed to background script (secure)
    ↓
Sent to sidebar component
    ↓
Displayed in Moly chat interface
    ↓
If user asks for suggestions:
    Sent to Claude API (HTTPS encrypted)
    ↓
Claude generates suggestions
    ↓
Suggestions displayed in sidebar
    ↓
Suggestions deleted after display (not stored)
    
NEVER:
✗ Stored on Moly servers
✗ Sent to third parties
✗ Logged or tracked
✗ Used for anything except message suggestions
```

---

## SUMMARY

**Moly's Message Detection Capability:**

✅ Automatically detects new messages on ANY website
✅ Works on Tinder, Facebook, LinkedIn, FetLife, Discord, etc.
✅ Reads message text without user copy-paste
✅ Identifies sender information automatically
✅ Detects platform and pre-selects appropriate context
✅ Shows message pre-filled in chat interface
✅ Seamless, automatic user experience
✅ Fallback options if detection fails
✅ Secure - no sensitive data exposed
✅ Efficient - minimal CPU/memory impact

**The user experience:**
1. User receives message on dating app/website
2. Moly detects automatically (no user action needed)
3. Message appears in Moly sidebar
4. User opens Moly chat
5. Message already visible, context pre-selected
6. User asks for suggestions or types response
7. Moly generates personalized suggestions
8. User sends via website (not through Moly)

---

**Version:** 1.0  
**Status:** Ready for Development  
**Last Updated:** August 4, 2026
