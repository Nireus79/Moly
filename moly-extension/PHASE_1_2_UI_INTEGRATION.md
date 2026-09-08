# Phase 1.2 UI Integration Guide

**Status**: Ready for Integration  
**Date**: September 8, 2026  
**Components**: ProfileView + ProfileAPI

---

## Overview

This guide explains how to integrate the Phase 1.2 profile display components into the existing Moly browser extension UI.

### What Was Added

1. **ProfileView.tsx** - React component displaying user profile with tabs
2. **profileAPI.ts** - API client for Phase 1.2 backend communication
3. **Sidebar integration** - Button to open profile view

### Key Features

✅ **Display Profile Data**
- AboutMe (communication style, tone, values)
- Patterns (observed behavioral patterns with confidence)
- Contacts (people tracked with communication patterns)
- Goals (communication goals with progress)
- Learnings (system-learned insights with confirm/reject)
- Reflections (user's journal entries)

✅ **User Interactions**
- Confirm/reject learnings
- Add reflections
- Update goal progress
- View confidence scores

✅ **Backend Integration**
- Connects to Phase 1.2 Go backend
- Uses X-User-ID header authentication
- Handles errors gracefully
- Offline detection

---

## File Locations

### New Files Created

```
moly-extension/src/
├── sidebar/components/
│   └── ProfileView.tsx          NEW - Profile display component
├── api/
│   └── profileAPI.ts            NEW - Phase 1.2 API client
└── sidebar/components/
    └── index.ts                 UPDATED - Export ProfileView
```

### Existing Files (No Changes Needed)

```
moly-extension/src/
├── sidebar/
│   ├── Sidebar.tsx              Main sidebar component
│   ├── components/
│   │   ├── ChatHistory.tsx
│   │   ├── MessageInput.tsx
│   │   ├── Suggestions.tsx
│   │   └── MeProfileModal.tsx
│   └── sidebar.css
├── stores/
│   ├── chatStore.ts
│   ├── settingsStore.ts
│   └── contactStore.ts
└── api/
    ├── backendManager.ts
    ├── molyAgent.ts
    └── providerManager.ts
```

---

## Integration Steps

### Step 1: Import ProfileView in Sidebar

In `src/sidebar/Sidebar.tsx`, add to imports:

```typescript
import { ..., ProfileView } from './components';
```

Already there in the component imports at line 5:
```typescript
import { ..., ReflectionModal, BackendStatus, SafetyAlert, MeProfileModal } from './components';
```

Just add `ProfileView` to this import.

### Step 2: Add State for Profile View

In the `Sidebar` component state section, add:

```typescript
const [showProfile, setShowProfile] = useState(false);
```

This goes with the existing state declarations around line 22-40.

### Step 3: Add Profile Button to UI

Add a button to open the profile view. Good location is in the top toolbar next to Settings:

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

### Step 4: Render ProfileView Component

At the end of the Sidebar return statement, before the closing tags, add:

```typescript
<ProfileView
  isOpen={showProfile}
  onClose={() => setShowProfile(false)}
  userId={currentUser?.id || ''}
  backendUrl={settings.backendUrl || 'http://localhost:8080'}
/>
```

### Step 5: Configure Backend URL in Settings

Update settings to include Phase 1.2 backend URL (if not already there):

In `src/stores/settingsStore.ts`, make sure settings have:

```typescript
backendUrl: string;  // URL of Phase 1.2 Go backend
```

Example: `http://localhost:8080` or `https://moly-backend.example.com`

---

## Usage Examples

### Using ProfileAPI Directly

```typescript
import { profileAPI } from '@/api/profileAPI';

// Set user ID
await profileAPI.setUserId('user123');

// Fetch profile
const profile = await profileAPI.getProfile();

// Get specific components
const patterns = await profileAPI.getPatterns();
const goals = await profileAPI.getGoals();

// User actions
await profileAPI.confirmLearning(42);
await profileAPI.addReflection(
  'Realized I avoid conflict in meetings',
  'breakthrough',
  ['communication', 'growth']
);
```

### Opening Profile View from Components

```typescript
// In any component that has access to state setters:
const [showProfile, setShowProfile] = useState(false);

return (
  <>
    <button onClick={() => setShowProfile(true)}>
      View My Profile
    </button>
    
    <ProfileView
      isOpen={showProfile}
      onClose={() => setShowProfile(false)}
    />
  </>
);
```

---

## Configuration

### Backend URL

The ProfileView component accepts `backendUrl` prop. Default is `http://localhost:8080`.

**For Development:**
```
http://localhost:8080
```

**For Production:**
```
https://api.moly.app/phase1.2
```

### User Identification

The extension needs to know which user to fetch data for. Options:

1. **From Chrome Storage** (recommended)
   ```typescript
   const result = await chrome.storage.local.get('userId');
   ```

2. **From Settings**
   ```typescript
   const userId = settings.userId;
   ```

3. **Passed as Prop**
   ```typescript
   <ProfileView userId={currentUser.id} />
   ```

---

## API Endpoints Used

The ProfileView component calls these Phase 1.2 endpoints:

```
GET  /api/v2.1/profile              - Complete profile
GET  /api/v2.1/profile/about-me     - Communication profile
GET  /api/v2.1/profile/contacts     - Contacts list
GET  /api/v2.1/profile/patterns     - Patterns
GET  /api/v2.1/profile/goals        - Goals
GET  /api/v2.1/profile/learnings    - Learnings
GET  /api/v2.1/profile/reflections  - Reflections
POST /api/v2.1/learnings/{id}/confirm
POST /api/v2.1/learnings/{id}/reject
```

All require `X-User-ID` header for authentication.

---

## Component Props

### ProfileView

```typescript
interface ProfileViewProps {
  isOpen: boolean;           // Show/hide the modal
  onClose: () => void;       // Callback when user closes modal
  userId?: string;           // User ID (optional, fetches from storage if not provided)
  backendUrl?: string;       // Backend URL (default: http://localhost:8080)
}
```

### Usage

```typescript
<ProfileView
  isOpen={showProfile}
  onClose={() => setShowProfile(false)}
  userId="user123"
  backendUrl="http://localhost:8080"
/>
```

---

## Features

### Overview Tab
- Shows AboutMe profile (communication style, tone, values)
- Displays confidence scores by category
- Shows extraction count

### Patterns Tab
- Lists observed communication patterns
- Shows observation count and confidence
- Marks growth areas
- Displays pattern category

### Contacts Tab
- Lists people mentioned in conversations
- Shows relationship type and mention frequency
- Displays communication patterns per contact
- Shows main topics discussed

### Goals Tab
- Lists communication goals
- Shows status (active/achieved/paused/abandoned)
- Displays progress notes
- Shows confidence level

### Learnings Tab
- Shows system-learned insights
- Color-coded by status (unreviewed/confirmed/rejected)
- Confirm/Reject buttons for unreviewed learnings
- Shows confidence and learning type

### Reflections Tab
- Displays user's journal entries
- Shows entry type (reflection/learning/breakthrough/struggle)
- Lists tags
- Shows entry content

---

## Styling

All components use inline styles for compatibility with extension environment. Colors follow this theme:

```
Blue:    #3b82f6  (primary actions)
Green:   #10b981  (confirm/success)
Red:     #ef4444  (reject/error)
Amber:   #f59e0b  (medium confidence)
Gray:    #6b7280  (neutral)
Light:   #f9fafb  (backgrounds)
```

---

## Error Handling

ProfileView handles errors gracefully:

1. **Network Error**: Shows error message
2. **Backend Unavailable**: Shows "Failed to load profile"
3. **Missing Data**: Shows "No X yet" messages
4. **Invalid User**: Falls back to fetching from storage

```typescript
if (error) {
  return <div>{error}</div>;
}
```

---

## Performance Considerations

### Data Loading

- Loads entire profile on open (all tabs in one request)
- Caches profile in component state
- Reload on confirm/reject actions

### Optimization Opportunities

1. **Lazy Load Tabs**: Load tab content only when tab is clicked
2. **Pagination**: For users with many reflections/learnings
3. **Background Sync**: Periodically refresh profile in background

---

## Testing

### Local Development

1. Start Phase 1.2 backend:
   ```bash
   cd moly-go
   go run main.go
   ```

2. Start extension:
   ```bash
   cd moly-extension
   npm run dev
   ```

3. Load extension in Chrome:
   - Go to `chrome://extensions/`
   - Enable Developer Mode
   - Click "Load unpacked"
   - Select `moly-extension/dist`

4. Test profile view:
   - Open sidebar
   - Click "Profile" button
   - Should display empty profile (no data yet)

### Manual Testing Checklist

- [ ] Profile loads without errors
- [ ] All tabs work (click through each)
- [ ] Confidence badges display correctly
- [ ] Confirm button works for learnings
- [ ] Reject button works for learnings
- [ ] Close button works
- [ ] Backend URL configuration works
- [ ] User ID detection works
- [ ] Error messages display properly

---

## Common Issues

### Profile Won't Load

**Problem**: Backend URL is wrong or backend not running

**Solution**:
1. Check backend is running: `http://localhost:8080/health`
2. Verify URL in settings
3. Check browser console (F12) for errors
4. Check backend logs

### User ID Not Found

**Problem**: Extension doesn't know which user to load

**Solution**:
1. Set userId in chrome storage:
   ```javascript
   chrome.storage.local.set({ userId: 'user123' });
   ```
2. Pass userId as prop to ProfileView
3. Check settings store for userId

### CORS Errors

**Problem**: Backend refusing requests from extension

**Solution**:
1. Ensure backend has proper CORS headers
2. Check backend logs for blocked requests
3. Verify X-User-ID header is being sent

---

## Future Enhancements

### v1 (Next Phase)
- [ ] Add reflection modal from profile view
- [ ] Edit goal directly from profile
- [ ] Export profile as PDF
- [ ] Share profile with contact (encrypted)

### v2 (Future)
- [ ] Real-time profile sync
- [ ] Collaborative profiles (team Moly)
- [ ] Profile history/timeline
- [ ] Advanced filtering and search
- [ ] Custom profile templates

---

## Integration Checklist

- [ ] Copy ProfileView.tsx to components/
- [ ] Copy profileAPI.ts to api/
- [ ] Update components/index.ts to export ProfileView
- [ ] Import ProfileView in Sidebar.tsx
- [ ] Add showProfile state
- [ ] Add Profile button to toolbar
- [ ] Render ProfileView component
- [ ] Test locally with backend running
- [ ] Verify all tabs work
- [ ] Test error handling
- [ ] Test user ID detection
- [ ] Test confirm/reject learning actions
- [ ] Documentation complete

---

## Next Steps

1. **Immediate**: Copy files and test integration locally
2. **Short Term**: Add reflection/goal management modals
3. **Medium Term**: Connect conversation analysis to profile
4. **Long Term**: Build full profile ecosystem

---

## Support

### Questions?
- Check browser console (F12) for errors
- Check backend logs for API errors
- Review PHASE_1_2_COMPLETE.md for backend documentation

### Backend Issues?
- Run Phase 1.2 backend: `go run main.go`
- Check port 8080 is available
- Verify database schema is deployed
- Check backend logs

---

**Created**: September 8, 2026  
**Status**: Ready for Implementation  
**Maintainer**: Claude Haiku 4.5
