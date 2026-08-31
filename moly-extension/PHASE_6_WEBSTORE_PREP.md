# Phase 6: Verification & Chrome Web Store Preparation

## Overview
Phase 6 prepares Moly for Chrome Web Store submission. This includes:
1. Verification that all features work correctly
2. Creation of Web Store assets
3. Preparation of submission materials
4. Pre-submission compliance check

---

## Step 1: Manual Verification

### 1.1 Quick Start
1. Build: `npm run build`
2. Go to `chrome://extensions/`
3. Enable "Developer mode" (top right)
4. Click "Load unpacked"
5. Select `dist/` folder
6. Extension should appear in toolbar

### 1.2 Sidebar Test
1. Click Moly extension icon
2. Sidebar should open in new tab
3. Verify no console errors (F12 → Console tab)

### 1.3 Settings Test
1. Click settings icon in sidebar header
2. Settings page should open
3. Try switching between providers (Claude, OpenAI, Ollama)
4. Settings page should load without errors

### 1.4 Provider Configuration Test
1. Get Claude API key from https://console.anthropic.com/keys
2. Enter key in Settings
3. Click "Save & Validate"
4. Should show "Configuration saved and validated successfully"
5. Can switch to OpenAI/Ollama to verify UI works

### 1.5 Chat Test
1. Return to sidebar
2. Type message in input area
3. Click Send (or Ctrl+Enter)
4. Message should appear in chat history
5. Wait for suggestions (3-10 seconds)
6. Should show 3-5 suggestions
7. Click Copy on any suggestion
8. Text should copy to clipboard
9. Should show "Copied!" feedback

---

## Step 2: Chrome Web Store Assets

### 2.1 Icon Assets Required

**Extension Icon (128x128)**
- Location: `public/icon-128.png`
- Format: PNG (transparent background preferred)
- Current: Probably need to design

**Extension Tile (440x280)**
- For Web Store listing
- Should be distinctive and represent the app

**Large Tile (920x680)**
- For featured placement
- Shows well on high resolution displays

### 2.2 Screenshots

**Screenshot 1: Main Interface** (1280x800)
- Show sidebar with sample chat
- Highlight chat input and suggestions area
- Include settings button in top right

**Screenshot 2: Settings Page** (1280x800)
- Show provider selection
- Highlight API key configuration
- Show "Save & Validate" button

**Screenshot 3: Suggestions** (1280x800)
- Show multiple suggestions displayed
- Highlight "Copy" button on suggestions
- Show "Copied!" feedback

**Screenshot 4: Communication Modes** (1280x800)
- Show chat mode selector (Socratic/Direct)
- Show context selector (Formal/Friendly/Dating)
- Highlight how settings affect suggestions

### 2.3 Text Content Required

**App Name** (45 char limit)
```
Moly - AI Messaging Assistant
```

**Short Description** (80 char limit)
```
Get AI-powered suggestions for better messages across any platform
```

**Detailed Description** (4000 char limit - see below)

**Detailed Description Template:**
```
Moly is your personal AI coaching assistant for crafting better messages.

KEY FEATURES:
- Chat with AI to build context about the person you're messaging
- Choose your vibe: Socratic (guiding questions) or Direct (ready-to-use suggestions)
- Set the tone: Formal, Friendly, or Dating
- Works across any website or messaging platform
- 100% local: Conversations stored privately on your device
- Multi-LLM: Use Claude, OpenAI, or Ollama (run locally)

HOW TO USE:
1. Open Moly sidebar
2. Tell Moly about the person you're messaging
3. Paste their message (optional) or just chat
4. Get AI-powered suggestions for your response
5. Copy and send the perfect message

SUPPORTED PROVIDERS:
- Claude: Anthropic's advanced AI (recommended)
- OpenAI: GPT-4 and GPT-3.5 Turbo
- Ollama: Run models locally, completely offline

PRIVACY:
- Zero tracking, zero analytics
- No data sent to Moly servers
- Only communicates with your chosen LLM provider
- Full conversation history stays on your device
- Clear all data anytime in settings

PRICING:
- Free tier: All features included
- Premium coming soon: Advanced features and priority support

Get better at messaging. Get Moly.
```

---

## Step 3: Web Store Submission Preparation

### 3.1 Create Developer Account
1. Go to https://chrome.google.com/webstore/developer/dashboard
2. Accept Developer Agreement
3. Pay $5 registration fee (one-time)
4. Verify email

### 3.2 Prepare Submission Package
1. Build extension: `npm run build`
2. Zip `dist/` folder
3. Ensure no git files included
4. Max file size: 10MB

### 3.3 Web Store Listing Fields

| Field | Value | Notes |
|-------|-------|-------|
| Name | Moly - AI Messaging Assistant | Max 45 chars |
| Short description | Get AI-powered suggestions... | Max 80 chars |
| Detailed description | [See template above] | Up to 4000 chars |
| Language | English | Default |
| Category | Productivity | Best fit |
| Icon | icon-128.png | 128x128 PNG |
| Screenshots | 4-5 screenshots | 1280x800 each |
| Locales | English | Can add more later |
| Support email | [Your email] | For user support |
| Support website | [Your website] | Optional link |

### 3.4 Privacy Policy & Permissions

**Privacy Policy** (REQUIRED)
- Created in: `moly-extension/docs/PRIVACY_POLICY.md`
- Must explain:
  - No personal data collection
  - No tracking/analytics
  - User controls all data flow
  - Local storage of conversations
  - Communication with LLM providers

**Permissions Justification**
- `storage`: Store conversations and settings locally
- `notifications`: (Optional) Notify user of generation complete
- NO host permissions: Cannot access websites

### 3.5 Content Policies Compliance

**Moly complies with:**
- No unauthorized access to user data
- No interference with website functionality
- No deceptive practices
- No adult content filtering/blocking
- Proper disclosure of AI assistance
- GDPR and privacy compliance

**Key Points for Submission:**
- Extension is READ-ONLY (user copies/pastes, we don't read automatically)
- Never sends messages automatically
- Never interferes with chat platforms
- User maintains full control
- Clear about AI assistance

---

## Step 4: Pre-Submission Checklist

### Code Quality
- [ ] No console errors
- [ ] No console warnings
- [ ] TypeScript builds without errors
- [ ] No hardcoded API keys
- [ ] No sensitive data in code

### Functionality
- [ ] Settings save and persist
- [ ] All providers can be configured
- [ ] Sidebar loads without errors
- [ ] Chat input/output works
- [ ] Suggestions generate and copy
- [ ] Dark mode CSS works (test in dark mode)

### UI/UX
- [ ] All text readable and clear
- [ ] No layout glitches
- [ ] Buttons are clickable
- [ ] Keyboard navigation works
- [ ] Mobile responsive (if applicable)

### Security
- [ ] No eval() or dangerous functions
- [ ] No CSP violations
- [ ] Secure API key handling
- [ ] No cross-site request forgery
- [ ] No injection vulnerabilities

### Documentation
- [ ] README.md complete
- [ ] User guide clear
- [ ] Privacy policy posted
- [ ] Compliance documentation ready
- [ ] CLAUDE.md up to date

### Assets
- [ ] Extension icon designed
- [ ] Screenshots created (4-5)
- [ ] Web Store copy written
- [ ] Thumbnail images prepared
- [ ] All images at correct resolution

---

## Step 5: Web Store Submission Process

### 5.1 Developer Dashboard
1. Login: https://chrome.google.com/webstore/developer/dashboard
2. Click "New Item"
3. Upload zipped `dist/` folder
4. Wait for initial validation (usually instant)

### 5.2 Fill in Details
1. **Developer Account**
   - Contact email
   - Support email
   - Developer website (optional)

2. **Listing Details**
   - Upload icon (128x128)
   - Enter app name
   - Enter short description
   - Enter detailed description
   - Select category (Productivity)

3. **Images**
   - Upload 4-5 screenshots (1280x800)
   - Create promo tile (920x680) if desired

4. **Language & Locales**
   - Set to English
   - Can add other languages later

5. **Privacy & Content**
   - Accept content policies
   - Link to privacy policy
   - Confirm no sensitive content

### 5.3 Submit for Review
1. Review all information
2. Accept policies
3. Pay fee ($if applicable)
4. Submit for review
5. Receive confirmation email

### 5.4 Review Process
- **Timeline**: 1-3 days typically
- **Automated checks**: Content, security, malware
- **Manual review**: Functionality, UI, compliance
- **Decision**: Approved or rejected with feedback

### 5.5 After Approval
1. Extension published to Web Store
2. Appears in searches
3. Shareable link provided
4. Can view analytics and reviews
5. Can push updates anytime

---

## Step 6: Post-Launch

### Monitoring
- Monitor for user reviews
- Track rating (aim for 4.5+)
- Watch for reported bugs
- Monitor installation numbers

### Updates
- Fix bugs as reported
- Add requested features
- Improve performance
- Add new LLM providers

### Analytics (from Web Store)
- Installation count
- Active users
- Rating/reviews
- Geographic distribution
- Browser versions

---

## Quick Submission Checklist

```
PRE-SUBMISSION:
[ ] Extension builds without errors
[ ] All features tested and working
[ ] No console errors in any tab
[ ] Settings persist correctly
[ ] LLM provider works (at least one)
[ ] Sidebar loads quickly
[ ] Suggestions generate and copy

ASSETS:
[ ] Extension icon created (128x128)
[ ] 4-5 screenshots at 1280x800
[ ] Web Store copy written and proofread
[ ] Privacy policy drafted
[ ] Support email configured

DEVELOPER ACCOUNT:
[ ] Created Chrome Web Store developer account
[ ] Paid $5 registration fee
[ ] Email verified
[ ] Developer info complete

SUBMISSION:
[ ] Built production version
[ ] Created deployment zip
[ ] All store fields filled out
[ ] Screenshots uploaded
[ ] Privacy policy linked
[ ] Policies accepted
[ ] Submitted for review

POST-LAUNCH:
[ ] Extension published
[ ] Landing page created (optional)
[ ] Social media announcement
[ ] Tracked first user feedback
[ ] Fixed any critical issues
[ ] Responded to initial reviews
```

---

## Rejection Recovery

If submission is rejected, typically due to:

1. **Functionality Issues**
   - Test the exact rejection scenario
   - Fix and resubmit

2. **Policy Violation**
   - Review specific policy mentioned
   - Adjust code/description
   - Resubmit with explanation

3. **Security Issues**
   - Review Chrome security guidelines
   - Fix issue
   - Request manual review if needed

4. **UX/Clarity Issues**
   - Improve descriptions
   - Update screenshots
   - Clarify AI assistance in copy
   - Resubmit

---

## Success Metrics

- Extension gets published
- First 10-50 users install
- Rating of 4.0+ after reviews
- No critical bug reports
- Support email responses < 24 hours

---

## Next Steps (Post-Launch)

1. **v2.1 Features** (2-3 weeks)
   - Keyboard shortcuts
   - Conversation templates
   - Advanced analytics

2. **v2.2 Premium** (4-5 weeks)
   - Premium tier ($2.99/month)
   - Advanced features
   - Priority support

3. **v3.0** (2-3 months)
   - Mobile app
   - Desktop client
   - API for developers

---

*Last Updated: 2026-09-01*
*Status: Ready for Phase 6*
