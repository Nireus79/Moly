# Phase 1.2 UI Integration - Quick Start

**Time to integrate**: 5-10 minutes  
**Complexity**: Low (copy-paste code)  
**Testing**: 2-3 minutes

---

## What You're Adding

A "Profile" button in the sidebar that displays:
- Your communication profile (AboutMe)
- Observed patterns with confidence scores
- People you talk to most
- Communication goals
- System learnings (confirm/reject them)
- Your reflection journal

---

## File Changes Required

### 1. Update `src/sidebar/components/index.ts`

**Change this:**
```typescript
export { MeProfileModal } from './MeProfileModal';
```

**To this:**
```typescript
export { MeProfileModal } from './MeProfileModal';
export { ProfileView } from './ProfileView';
```

### 2. Update `src/sidebar/Sidebar.tsx`

**At line 5, change:**
```typescript
import { ChatHistory, MessageInput, Suggestions, SuggestionsV2, SettingsPanel, ConversationSelector, NewConversationModal, ContactManager, ReflectionModal, BackendStatus, SafetyAlert, MeProfileModal } from './components';
```

**To:**
```typescript
import { ChatHistory, MessageInput, Suggestions, SuggestionsV2, SettingsPanel, ConversationSelector, NewConversationModal, ContactManager, ReflectionModal, BackendStatus, SafetyAlert, MeProfileModal, ProfileView } from './components';
```

**Add state at line 40 (after existing state declarations):**
```typescript
const [showProfile, setShowProfile] = useState(false);
```

**Add button to UI toolbar. Find where other buttons like Settings are and add:**
```typescript
<button
  onClick={() => setShowProfile(true)}
  style={{
    padding: '8px 12px',
    background: '#3b82f6',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: '500',
    marginRight: '8px',
  }}
>
  📊 Profile
</button>
```

**At the very end of the return statement (before closing tags), add:**
```typescript
<ProfileView
  isOpen={showProfile}
  onClose={() => setShowProfile(false)}
/>
```

---

## Files to Copy

Copy these files from this repo to your extension:

```bash
# From moly-go/moly-extension/src/
cp src/sidebar/components/ProfileView.tsx ../moly-extension/src/sidebar/components/
cp src/api/profileAPI.ts ../moly-extension/src/api/
```

Or manually create:
1. `src/sidebar/components/ProfileView.tsx` - Full component (580 lines)
2. `src/api/profileAPI.ts` - API client (340 lines)

---

## Testing

### Backend Must Be Running

```bash
cd Moly/moly-go
go run main.go
```

You should see:
```
[Database] Initialized at ./moly.db (encrypted)
[Server] Starting on :8080
```

### Start Extension

```bash
cd Moly/moly-extension
npm run dev
```

### Load in Chrome

1. Open `chrome://extensions/`
2. Enable "Developer mode" (top right)
3. Click "Load unpacked"
4. Select `moly-extension/dist/`
5. Open extension
6. Click "📊 Profile" button
7. Should show empty profile or load data if any exists

---

## Verify It Works

**Checklist:**
- [ ] Button appears in sidebar
- [ ] Clicking button opens modal
- [ ] Modal shows "Profile" title
- [ ] "Overview" tab shows AboutMe section
- [ ] "Patterns" tab lists patterns (or empty)
- [ ] "Contacts" tab lists contacts (or empty)
- [ ] "Goals" tab lists goals (or empty)
- [ ] "Learnings" tab shows learnings (or empty)
- [ ] "Reflections" tab shows journal entries (or empty)
- [ ] Confirm/Reject buttons work for learnings
- [ ] Close button (X) closes the modal
- [ ] No console errors (F12 to check)

---

## Configuration

### Backend URL

By default, ProfileView looks for backend at `http://localhost:8080`

To change, pass as prop:

```typescript
<ProfileView
  isOpen={showProfile}
  onClose={() => setShowProfile(false)}
  backendUrl="https://api.moly.app/phase1.2"
/>
```

Or configure in settings and pass from there:

```typescript
<ProfileView
  isOpen={showProfile}
  onClose={() => setShowProfile(false)}
  backendUrl={settings.backendUrl || 'http://localhost:8080'}
/>
```

### User ID

ProfileView auto-detects user ID from Chrome storage:

```typescript
const result = await chrome.storage.local.get('userId');
// Uses result.userId if available
```

To set it:

```javascript
// In browser console
chrome.storage.local.set({ userId: 'your_user_id' })
```

---

## Troubleshooting

### Button doesn't appear
- Verify import was added
- Verify state was added
- Verify ProfileView is rendered at end
- Check browser console for errors

### Profile won't load
- Is backend running? `curl http://localhost:8080/health`
- Check browser console (F12) for error messages
- Check backend logs for request errors

### User ID not found
```javascript
// Set it manually in browser console
chrome.storage.local.set({ userId: 'test_user' })
```

### Style looks wrong
- ProfileView uses inline styles (works everywhere)
- If buttons look weird, check CSS conflicts
- Try refreshing extension (F12 → refresh)

---

## What Data Shows

### If Backend Has Data

Example after conversations have been analyzed:

**AboutMe Tab:**
- Communication Style: "conflict-avoidant"
- Tone Preference: "diplomatic"
- Core Values: "harmony, respect, authenticity"
- Confidence: 75%

**Patterns Tab:**
- avoids_conflict (observed 5 times, 85% confidence)
- passive_communication (observed 3 times, 70% confidence)
- people_pleaser (observed 4 times, 78% confidence)

**Contacts Tab:**
- Boss (mentioned 8 times)
- Mom (mentioned 6 times)
- Best Friend (mentioned 12 times)

**Goals Tab:**
- "Be more assertive in meetings" (active, 80% confidence)
- "Speak up when I disagree" (active, 70% confidence)

**Learnings Tab:**
- "Conflict-avoidant pattern detected" (unconfirmed)
- "Values harmony over honesty" (unconfirmed)

**Reflections Tab:**
- "Had breakthrough realizing I avoid conflict because I fear rejection"

### If No Data Yet

All tabs show: "No X yet"

This is normal on first run. Data builds as you use the extension.

---

## Next Steps

1. **Complete**: Copy files and integrate into Sidebar
2. **Test**: Verify button works and loads data
3. **Customize**: Add to settings page for URL configuration
4. **Enhance**: Add reflection/goal management modals

---

## Code Snippets Ready to Paste

### Import Line
```typescript
export { ProfileView } from './ProfileView';
```

### State
```typescript
const [showProfile, setShowProfile] = useState(false);
```

### Button
```typescript
<button
  onClick={() => setShowProfile(true)}
  style={{
    padding: '8px 12px',
    background: '#3b82f6',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: '500',
  }}
>
  📊 Profile
</button>
```

### Component
```typescript
<ProfileView
  isOpen={showProfile}
  onClose={() => setShowProfile(false)}
/>
```

---

## Full Example

Here's a minimal complete integration:

```typescript
import React, { useState } from 'react';
import { ProfileView } from './components';

export const Sidebar: React.FC = () => {
  const [showProfile, setShowProfile] = useState(false);

  return (
    <>
      <div style={{ padding: '10px' }}>
        <button onClick={() => setShowProfile(true)}>
          📊 Profile
        </button>
      </div>

      <ProfileView
        isOpen={showProfile}
        onClose={() => setShowProfile(false)}
      />
    </>
  );
};
```

---

## Support

- Full docs: `PHASE_1_2_UI_INTEGRATION.md`
- Backend docs: `../moly-go/PHASE_1_2_COMPLETE.md`
- Issues? Check browser console (F12)

---

**Integration Time**: 5 minutes  
**Testing Time**: 5 minutes  
**Total**: ~10 minutes to working profile view

**You're adding one of the key Phase 1.2 features: User can see their profile data, confirm/reject learnings, and review their communication patterns.**
