# Moly Privacy Policy

**Effective Date**: 2026-08-31  
**Version**: 2.0 (Redesigned)

---

## Overview

Moly is a personal messaging coach extension that operates entirely offline. **We do not collect, store, or transmit any personal data.**

This privacy policy explains how Moly handles data.

---

## Core Privacy Principle

**Everything stays on your device.**

- No servers
- No cloud storage
- No analytics
- No tracking
- No ads
- No third-party sharing

---

## What Data Moly Handles

### Data Moly Processes (Locally)
1. **Messages you paste into Moly**
   - Stored in your browser's local storage (chrome.storage.local)
   - Analyzed locally using your configured LLM provider
   - Never leaves your device unless you send it to LLM provider

2. **Conversation history**
   - Your messages and Moly's suggestions
   - Stored locally in your browser
   - Only visible to you
   - Deleted when you clear browser data or uninstall extension

3. **Settings and preferences**
   - Chat mode selection (Socratic/Direct)
   - Communication context (Formal/Friendly/Dating)
   - LLM provider configuration
   - Stored locally in your browser

### Data Moly Does NOT Collect
- ❌ Your identity, email, or account information
- ❌ Information from the websites you visit
- ❌ Messages from chat platforms
- ❌ Metadata about your browsing
- ❌ Location, device ID, or hardware info
- ❌ Usage analytics or telemetry

---

## LLM Provider Data Handling

**IMPORTANT**: When you use an LLM provider (Claude, OpenAI, Ollama), data flows as follows:

### If Using Claude (Anthropic)
- You provide your own API key
- Messages you paste into Moly are sent to Claude for analysis
- Claude's Privacy Policy applies: https://www.anthropic.com/privacy
- Anthropic may retain messages for safety/improvement purposes (per their policy)
- **Moly never stores Claude's responses** beyond what you see

### If Using OpenAI
- You provide your own API key
- Messages you paste are sent to OpenAI for analysis
- OpenAI's Privacy Policy applies: https://openai.com/privacy
- OpenAI may use API data per their policy
- **Moly never stores OpenAI's responses** beyond what you see

### If Using Ollama (Local)
- All processing happens on your device
- Zero data transmission
- Completely private
- **Recommended for maximum privacy**

---

## What Gets Stored Locally

### Chrome Storage (Local)
```javascript
// Example storage structure (visible in DevTools)
{
  "settings": {
    "activeProvider": "claude",
    "chatMode": "socratic",
    "defaultContext": "friendly"
  },
  "chatHistory": [
    {
      "userMessage": "[text you pasted]",
      "suggestions": ["suggestion 1", "suggestion 2"],
      "mode": "socratic",
      "context": "friendly",
      "timestamp": 1234567890
    }
  ]
}
```

**Storage Details:**
- Maximum ~5-10MB per extension
- Stored in your browser profile
- Deleted when you clear browser data
- Not synced across devices
- Not accessible to any website

---

## Data Deletion

### Automatic Deletion
- **Uninstall extension**: All data immediately deleted
- **Clear browser data**: All Moly data deleted
- **Browser profile deletion**: All Moly data deleted

### Manual Deletion
- **Settings → Clear All Data**: Delete all stored conversations and preferences
- **Individual history**: Click trash icon on any saved suggestion/message pair

---

## GDPR Compliance

Moly is **GDPR compliant** because:

✓ **No personal data collection**  
✓ **No data transmission**  
✓ **No profiling or tracking**  
✓ **Local processing only**  
✓ **User complete control**  
✓ **Easy data deletion**  
✓ **No third-party sharing**  

**If you use an LLM provider (Claude, OpenAI):**
- You are responsible for their data handling
- Review their privacy policies
- Their servers may be in different jurisdictions
- Consider using Ollama (local) for full GDPR compliance

---

## Security

### How Moly Protects Your Data

1. **No network transmission** (except to LLM providers)
   - Browser storage is isolated by origin
   - Not accessible to websites or other extensions
   - Cannot be intercepted in transit (because no transmission)

2. **No server vulnerabilities**
   - No servers = no server breaches
   - No cloud infrastructure = no cloud compromises
   - Local-only = local-only risk

3. **Browser isolation**
   - Data isolated in browser profile
   - Protected by browser security model
   - Password-protected profiles add protection

4. **No tracking or malware**
   - No external dependencies
   - No ads or trackers
   - Source code auditable (open source)

### Your Responsibility

- **Secure your device** (antivirus, firewall, updates)
- **Secure your browser** (don't share computer with untrusted users)
- **Protect API keys** (never share your Claude/OpenAI API keys)
- **Use Ollama locally** (for maximum privacy, no API key transmission)

---

## Third-Party Services

### Services Moly Uses
- **LLM Providers**: Claude (Anthropic), OpenAI, Ollama
  - You provide your own credentials
  - Direct API calls from your browser
  - See each provider's privacy policy

### Services Moly Does NOT Use
- ❌ Google Analytics
- ❌ Sentry or error tracking
- ❌ Mixpanel or data analytics
- ❌ Advertising networks
- ❌ Social media tracking
- ❌ CDNs for user data
- ❌ Cloud storage services

---

## Platform Integration

Moly **does not**:
- ❌ Read chat platform content automatically
- ❌ Access platform APIs
- ❌ Store platform credentials
- ❌ Interfere with platform functionality
- ❌ Modify platform UI

Moly **only**:
- ✓ Analyzes text you explicitly paste into Moly
- ✓ Processes data locally
- ✓ Returns suggestions you can manually copy

---

## Changes to Privacy Policy

We may update this policy to reflect new features or regulatory changes.

**You will be notified of material changes via:**
- Extension update notification
- Privacy Policy update in settings

**No retroactive changes** - policy changes apply only to new data collection going forward.

---

## Your Rights

You have the right to:
- ✓ Access your stored data (Settings → Data)
- ✓ Delete your data (Settings → Clear All)
- ✓ Export your conversation history
- ✓ Use without tracking or profiling
- ✓ Complete privacy and transparency

---

## Contact & Questions

**Questions about this Privacy Policy?**
- Email: efthimiosangelopoulos@gmail.com
- GitHub Issues: (if open-sourced)

---

## Summary

**Moly's Privacy Commitment:**

| Aspect | Moly's Approach |
|--------|-----------------|
| **Data Collection** | None (only what you manually input) |
| **Data Storage** | Local only (on your device) |
| **Data Transmission** | Only to your chosen LLM provider |
| **Third-party Sharing** | Never |
| **Analytics** | None |
| **Tracking** | None |
| **Profiling** | None |
| **User Control** | Complete |
| **Deletion** | Instant and complete |

**Bottom line**: Moly is a tool that respects your privacy completely. We don't collect, store, or share any data about you. Everything happens on your device.
