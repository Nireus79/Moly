# Moly Privacy & Security Specification

**Date**: September 6, 2026  
**Status**: Security Reference  
**Version**: 1.0

---

## Core Privacy Principle

**Moly monitors USER behavior, NEVER contact behavior.**

This is enforced at every layer: database constraints, code reviews, architecture decisions.

---

## What Moly Collects

### ✅ USER BEHAVIOR (Collected & Analyzed)

**What we learn**:
- How user communicates (tone, length, emoji usage, directness)
- Communication goals (opening messages, celebrations, apologies)
- Suggestion preferences (which tones they pick, what edits they make)
- Success patterns (what suggestions lead to positive contact responses)
- Personality traits (inferred from language and choices)
- Growth over time (increasing confidence, skill development)
- Risk patterns (manipulation tendencies, boundary issues)

**How we use it**:
- Personalize suggestions to match user's authentic style
- Learn what interventions work for this user
- Detect concerning patterns and educate
- Improve over time with behavioral learning

**Stored in**:
- `user_profiles` table
- `interaction_history` table
- `risk_patterns` table

---

### ❌ CONTACT BEHAVIOR (Never Collected)

**What we DON'T track**:
- How contacts respond
- Contact's message patterns
- Contact's communication style
- Contact's response times
- Contact's availability
- Contact's personality traits
- Any monitoring of the contact

**Why not**:
- It's surveillance without consent
- Contact data belongs to user, not Moly
- Violates privacy principle
- Enables creepy features we reject

**Technical enforcement**:
- No database columns for contact monitoring
- Database constraints prevent contact data storage
- Code reviews verify no contact surveillance
- Risk Monitoring Agent only tracks USER patterns

---

## Data Minimization

### Only Store What's Necessary

**About Me**: User's own description of themselves
```json
{
  "communicationStyle": "casual, direct, authentic",
  "values": ["authenticity", "loyalty"],
  "notes": "I'm an introvert but love deep conversations"
}
```

**Contact**: User's observations only
```json
{
  "name": "Sarah",
  "characteristics": ["ambitious", "direct"],
  "interests": ["tech", "hiking"],
  "notes": "Software engineer, career-focused",
  "communicationPreferences": "No small talk, warm in personal"
}
```

**NOT stored**:
- "Sarah responds to messages within 2 hours"
- "Sarah prefers formal communication"
- "Sarah has been busier lately"
- "Sarah usually initiates conversations"

---

## Data Access & Authorization

### User Access Rights

Users can:
- View their own About Me ✅
- View their own Contacts ✅
- View their own Conversations ✅
- View their own Behavioral Profile ✅
- Export all their data ✅
- Delete their account and all data ✅

Users cannot:
- View contact's data ❌ (doesn't exist)
- View contact's profile ❌ (only their observations)
- Access other users' data ❌

### Backend Access

Only the following agents/components can access data:
- **Conversation Agent**: Reads context to generate suggestions
- **Learning Agent**: Reads user behavior to build profile
- **Context Manager**: Reads/writes About Me, Contacts, Conversations
- **Risk Monitoring Agent**: Reads user's risk patterns

**Cannot access**:
- Other users' data (isolated by user_id)
- Contact surveillance data (doesn't exist)
- Raw API keys (stored encrypted)

### API Authentication

Future: JWT tokens with scopes
```
GET /api/context (requires: read_context)
POST /api/conversation/generate (requires: read_context, write_interaction)
DELETE /api/user (requires: delete_account)
```

---

## Encryption

### In Transit

- **HTTPS/TLS 1.3** for all communication
- Certificate pinning in extension (future)

### At Rest

**Sensitive data encrypted**:
```
- API keys: AES-256-GCM
- User behavioral data: Optional (depends on trust)
```

**Configuration**:
```go
// encryption/keys.go
type KeyManager struct {
  masterKey []byte // Loaded from secure vault
  rotationInterval time.Duration = 90 * 24 * time.Hour
}

func (k *KeyManager) Encrypt(plaintext []byte) ([]byte, error) {
  cipher, _ := aes.NewCipher(k.masterKey)
  gcm, _ := cipher.NewGCM()
  nonce := make([]byte, gcm.NonceSize())
  io.ReadFull(rand.Reader, nonce)
  return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

---

## Data Retention

### Permanent (User can delete)
- About Me
- Contacts
- Conversation history (user's choice)

### Retention Limits
- User behavioral profile: Keep indefinitely (user can request deletion)
- Risk patterns: Keep indefinitely
- Interaction history: 1 year (archive then delete)
- Audit logs: 2 years for compliance

### User Deletion (Right to be Forgotten)

When user requests account deletion:
```sql
-- Delete all user data
BEGIN TRANSACTION;
  DELETE FROM reflections WHERE user_id = ?;
  DELETE FROM messages WHERE conversation_id IN 
    (SELECT id FROM conversations WHERE user_id = ?);
  DELETE FROM conversations WHERE user_id = ?;
  DELETE FROM contacts WHERE user_id = ?;
  DELETE FROM about_me WHERE user_id = ?;
  DELETE FROM interaction_history WHERE user_id = ?;
  DELETE FROM risk_patterns WHERE user_id = ?;
  DELETE FROM user_profiles WHERE user_id = ?;
  DELETE FROM users WHERE id = ?;
COMMIT;

-- Keep audit log for compliance
INSERT INTO audit_log (action, entity_type, reason)
  VALUES ('account_deleted', 'user', 'GDPR right to be forgotten');
```

---

## Compliance

### GDPR (EU)

**Requirements**:
- ✅ Data minimization: Only collect necessary data
- ✅ Purpose limitation: Use data only for personalization
- ✅ Storage limitation: 1-2 year retention
- ✅ User rights: Access, export, delete data
- ✅ Consent: Clear opt-in (set API key)
- ✅ Breach notification: Report within 72 hours
- ✅ Privacy by design: Built into architecture

**Implementation**:
```go
// compliance/gdpr.go
type GDPRDataExport struct {
  AboutMe Interface
  Contacts []Contact
  Conversations []Conversation
  BehavioralProfile UserProfile
  ExportDate timestamp
  ExportedBy string // User or admin
}

func (u *User) ExportAllData() ([]byte, error) {
  export := GDPRDataExport{ ... }
  return json.Marshal(export)
}
```

### CCPA (California)

**Requirements**:
- ✅ Right to know: Users can request data
- ✅ Right to delete: "Delete me" button
- ✅ Right to opt-out: Can stop learning
- ✅ No discrimination: Same service without data collection

### HIPAA (if applicable)

- Only if users share health data
- Current: Not applicable
- Future: If mental health features added

---

## Third-Party Access

### What Data Leaves Moly

**To Claude API**:
- User's message (for processing)
- Contact profile (for context)
- About Me (for personalization)
- Conversation history (for context)

**NOT sent**:
- Raw API keys
- User's contact surveillance data (doesn't exist)
- Behavioral profile (stays local)

**Configuration**:
```go
// api/claude.go
func (c *ClaudeClient) GenerateSuggestions(ctx Context) {
  // What we send
  sanitized := SanitizeForClaude(ctx)
  
  // Strip sensitive data
  sanitized.UserProfile = nil // Don't send behavioral profile
  sanitized.ContactSurveillance = nil // Never send (it's null anyway)
  
  // Send only what's needed
  response, _ := c.API.Call(sanitized)
}
```

### Data Processing Agreements

If using third-party services:
- ✅ Data Processing Addendum (DPA) signed
- ✅ Sub-processor agreements
- ✅ Data residency (EU data stays in EU)
- ✅ Encryption in transit

---

## Security Best Practices

### API Key Security

**Extension**:
```typescript
// Never store API key in localStorage
// Always use chrome.storage.sync with encryption
chrome.storage.sync.set({
  apiKey: encrypt(apiKeyValue)
});

// Never log API keys
console.log(`API Key: ${apiKey}`); // NEVER
logger.info('API key configured'); // OK
```

**Backend**:
```go
// Never log API keys
logger.Debugf("API Key: %s", apiKey) // NEVER
logger.Info("Claude API configured") // OK

// Hash API keys for comparison
stored := hashAPIKey(providedKey)
if stored == hashAPIKey(userKey) {
  // Keys match
}
```

### SQL Injection Prevention

```go
// Always use parameterized queries
// WRONG:
query := "SELECT * FROM users WHERE id = " + userId // VULNERABLE

// RIGHT:
var user User
db.QueryRow("SELECT * FROM users WHERE id = ?", userId).Scan(&user)
```

### CORS Configuration

```go
// WRONG:
router.Use(cors.AllowedOrigins("*")) // Opens to all sites

// RIGHT:
router.Use(cors.AllowedOrigins(
  "chrome-extension://moly-extension-id",
  "chrome-extension://moly-extension-id-2", // Other users' installs
))
```

### Rate Limiting & DDoS Protection

```go
// Prevent abuse
limiter := NewRateLimiter(
  requestsPerMinute: 100,
  perUser: true,
  burstSize: 10,
)

if !limiter.Allow(userID) {
  return 429, "Rate limited"
}
```

### Dependency Scanning

```bash
# Regular security audits
go list -json -m all | nancy sleuth

# Weekly scanning
npm audit
go mod graph | nancy

# SBOM (Software Bill of Materials)
syft generate -o cyclonedx > sbom.json
```

---

## Incident Response Plan

### Privacy Breach

**If user data compromised**:
1. Identify scope: Which users? Which data?
2. Contain: Revoke tokens, force password reset
3. Notify: Email users within 72 hours (GDPR)
4. Investigate: Root cause analysis
5. Fix: Apply patch, update systems
6. Document: Incident report, lessons learned

**Checklist**:
- [ ] Backup/preserve evidence
- [ ] Notify legal team
- [ ] Contact Anthropic
- [ ] Audit logs
- [ ] Customer notification
- [ ] Regulatory reporting
- [ ] Post-mortem

### Security Incident

**If system hacked**:
1. Isolate: Take system offline
2. Investigate: What accessed? When? How?
3. Secure: Patch vulnerability, rotate keys
4. Restore: From clean backup
5. Verify: Security scanning
6. Notify: Users of any exposed data

---

## Security Checklist

**Before Production**:
- [ ] All API keys in environment, not code
- [ ] Database password in secret manager
- [ ] HTTPS/TLS enabled
- [ ] CORS configured correctly
- [ ] SQL injection prevention verified
- [ ] XSS protection headers
- [ ] CSRF tokens if applicable
- [ ] Rate limiting enabled
- [ ] Input validation on all endpoints
- [ ] Security headers configured
- [ ] Dependency scanning in CI/CD
- [ ] Secrets scanning in CI/CD
- [ ] SAST (static analysis) passed
- [ ] Penetration testing completed

**In Production**:
- [ ] Daily security log review
- [ ] Weekly patch management
- [ ] Monthly dependency updates
- [ ] Quarterly penetration testing
- [ ] Annual security audit
- [ ] Incident response plan tested

---

## Privacy Policy (User-Facing)

**Summary**:

Moly learns about YOUR communication style to personalize suggestions. We never monitor or track your contacts. Your data belongs to you.

**What we collect**:
- How you communicate (tone, language, style)
- What you choose to do with our suggestions
- Your About Me profile (in your own words)
- Your contact list (your observations only)

**What we don't collect**:
- How your contacts respond
- Your contacts' communication style
- Any monitoring of your relationships
- Any surveillance data

**Your rights**:
- Access all your data
- Download your data (export)
- Delete your account and all data
- Opt-out of learning (but features limited)

**Questions?**
- Privacy: privacy@moly.dev
- Security: security@moly.dev
- Deletion: hello@moly.dev

---

## Ethical Guidelines

### What Moly Believes

1. **User Autonomy**: Users decide how to communicate. We advise; they choose.
2. **No Surveillance**: We learn from user behavior, never monitor contacts.
3. **Transparency**: We explain what we know and why.
4. **Education > Enforcement**: We teach ethics, not dictate.
5. **Privacy First**: Minimize data, maximize control.

### What Moly Rejects

1. ❌ **Manipulation**: Using data to trick users or contacts
2. ❌ **Surveillance**: Monitoring contacts without consent
3. ❌ **Deception**: Hidden tracking or data sales
4. ❌ **Coercion**: Forcing choices through dark patterns
5. ❌ **Discrimination**: Treating users differently based on profile

---

This specification ensures Moly is built with privacy and security as core features, not afterthoughts.
