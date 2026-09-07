# Moly API Specification - Complete Reference

**Date**: September 6, 2026  
**Status**: API Reference  
**Version**: 1.0

---

## Overview

All communication between extension and backend uses JSON over HTTPS. Requests are stateless; all context comes in the request body.

**Base URL**: `http://localhost:11436` (auto-detected or configured)

---

## Authentication

**No explicit auth currently.** Future: Add JWT tokens in `Authorization: Bearer <token>` header.

---

## Core Endpoints

### 1. Generate Conversation Response

**Endpoint**: `POST /api/conversation/generate`

**Purpose**: Main orchestration endpoint. Runs Conversation Agent.

**Request**:
```json
{
  "conversationId": "conv_abc123",
  "userId": "user_xyz789",
  "userMessage": "Sarah just got promoted and I don't know what to say",
  "mode": "direct",
  "tone": "friendly",
  "metadata": {
    "type": "generate",
    "timestamp": 1725609600000,
    "previousSuggestionChosen": null,
    "userModified": null,
    "contactResponse": null,
    "userFeedback": null
  }
}
```

**Response** (Success - 200):
```json
{
  "phase": "suggestions_ready",
  "suggestions": [
    {
      "index": 0,
      "text": "That's amazing! I'm so proud of you. Let's celebrate soon!",
      "tone": "friendly",
      "reasoning": "Celebrates achievement, genuine, matches your authentic style",
      "confidence": 0.92
    },
    {
      "index": 1,
      "text": "Congrats! You totally deserved it. I'd love to hear all about it.",
      "tone": "friendly",
      "reasoning": "Warm, interested, avoids making it about you",
      "confidence": 0.88
    },
    {
      "index": 2,
      "text": "That's incredible! You must be thrilled. Free this weekend to celebrate?",
      "tone": "friendly",
      "reasoning": "Celebratory, action-oriented, matches your style",
      "confidence": 0.85
    }
  ],
  "questions": [
    "What does celebrating look like for you two?",
    "Is she taking time off to enjoy this?"
  ],
  "reflection": {
    "id": "refl_123",
    "conversationId": "conv_abc123",
    "characteristics": ["ambitious", "achievement-focused"],
    "interests": ["career", "growth"],
    "communicationPreferences": "Direct, celebrates openly",
    "intentions": ["celebrate achievement", "deepen connection"],
    "status": "pending_approval",
    "userQuotes": ["just got promoted", "I'm so proud"]
  },
  "riskWarning": null,
  "safetyAlert": null,
  "constitutionConcerns": null,
  "processingTimeMs": 1247,
  "metadata": {
    "agentsUsed": ["conversation", "learning", "context_manager"],
    "backendVersion": "1.0.0"
  }
}
```

**Response** (Safety Alert - 200):
```json
{
  "phase": "safety_alert",
  "safetyAlert": {
    "alert_type": "crisis",
    "severity": "immediate",
    "title": "Crisis Detected",
    "message": "Your message suggests you might be in crisis.",
    "indicators": ["suicidal language", "self-harm"],
    "resources": [
      {
        "name": "National Suicide Prevention Lifeline",
        "description": "24/7 support",
        "number": "988",
        "url": "https://988lifeline.org",
        "region": "US"
      }
    ],
    "recommendations": ["Call 988", "Text HOME to 741741", "Go to nearest ER"]
  },
  "suggestions": [],
  "reflection": null
}
```

**Response** (Risk Warning - 200):
```json
{
  "phase": "risk_warning",
  "riskWarning": {
    "riskLevel": "high",
    "pattern": "manipulation",
    "severity": 8,
    "educationalQuestions": [
      "I'm noticing this focuses on making yourself look good. Why does that matter?",
      "How do you think she'd feel if you made it about yourself?",
      "What if you just celebrated her achievement instead?"
    ],
    "principles": [
      {
        "id": "authenticity",
        "name": "Authenticity",
        "description": "Be genuine, not strategic"
      }
    ],
    "alternatives": [
      "Just celebrate her achievement genuinely",
      "Ask her how she feels about it",
      "Share in her joy without comparing yourself"
    ],
    "recommendation": "educate_first",
    "message": "I noticed something. Help me understand why you want to mention that..."
  },
  "suggestions": [
    // Still provided, user not blocked
  ],
  "reflection": null
}
```

**Response** (Error - 400-500):
```json
{
  "phase": "error",
  "error": "Invalid request: missing conversationId",
  "statusCode": 400,
  "timestamp": 1725609600000,
  "requestId": "req_xyz"
}
```

---

### 2. Submit Conversation Feedback

**Endpoint**: `POST /api/conversation/feedback`

**Purpose**: Record user's choice after suggestions shown.

**Request**:
```json
{
  "conversationId": "conv_abc123",
  "userId": "user_xyz789",
  "suggestionChosen": 0,
  "suggestionText": "That's amazing! I'm so proud of you...",
  "userModified": false,
  "modificationRequest": null,
  "reflectionApproved": true,
  "reflectionEdits": {
    "characteristics": ["ambitious", "achievement-focused"],
    "interests": ["career", "growth", "hiking"],
    "communicationPreferences": "Direct, celebrates openly",
    "userNotes": "Added: loves hiking"
  },
  "timestamp": 1725609800000
}
```

**Response** (Success - 200):
```json
{
  "status": "recorded",
  "message": "Thank you for the feedback. Learning updated.",
  "updatedContact": {
    "id": "contact_123",
    "name": "Sarah",
    "characteristics": ["ambitious", "achievement-focused"],
    "interests": ["career", "growth", "hiking"],
    "notes": "Direct communicator, celebrates openly. Loves hiking.",
    "updatedAt": 1725609800000
  },
  "learningUpdated": {
    "userProfileVersion": 5,
    "contactProfileVersion": 3
  }
}
```

---

### 3. Get Context

**Endpoint**: `GET /api/context?conversationId=conv_abc123&userId=user_xyz789`

**Purpose**: Retrieve relevant context for a conversation (About Me, Contact, History).

**Response** (Success - 200):
```json
{
  "aboutMe": {
    "userId": "user_xyz789",
    "communicationStyle": "casual, direct, authentic",
    "values": ["authenticity", "loyalty", "growth", "fun"],
    "preferredTone": "friendly",
    "notes": "I'm an introvert but love deep conversations",
    "updatedAt": 1725609600000
  },
  "contactProfile": {
    "id": "contact_123",
    "name": "Sarah",
    "relationship": "close_friend",
    "characteristics": ["ambitious", "direct", "achievement-focused"],
    "interests": ["tech", "hiking", "coffee"],
    "communicationPreferences": "No small talk, warm in personal matters",
    "notes": "Software engineer, very career-focused",
    "updatedAt": 1725609600000
  },
  "conversationHistory": [
    {
      "role": "user",
      "content": "Tell me about Sarah",
      "type": "message",
      "timestamp": 1725608000000
    },
    {
      "role": "assistant",
      "content": "Tell me about her. What's she like?",
      "type": "question",
      "timestamp": 1725608100000
    },
    {
      "role": "user",
      "content": "She's into tech, very direct, no-nonsense",
      "type": "message",
      "timestamp": 1725608200000
    }
  ],
  "userBehaviorProfile": {
    "communicationStyle": "casual, direct",
    "preferredTone": "friendly",
    "tonePreferences": {
      "formal": 0.15,
      "friendly": 0.70,
      "dating": 0.15
    },
    "suggestionChoiceRate": 0.68,
    "emergingPersonality": ["values authenticity", "prefers casual"],
    "growthTrend": "increasing_confidence"
  },
  "contextQuality": "complete",
  "gaps": [],
  "timestamp": 1725609600000
}
```

---

### 4. Get Conversations

**Endpoint**: `GET /api/conversations?userId=user_xyz789`

**Purpose**: List all conversations for user.

**Response** (Success - 200):
```json
{
  "conversations": [
    {
      "id": "conv_abc123",
      "userId": "user_xyz789",
      "contactId": "contact_123",
      "contactName": "Sarah",
      "messageCount": 12,
      "lastMessage": "That sounds great!",
      "lastMessageTime": 1725609500000,
      "createdAt": 1725605000000
    },
    {
      "id": "conv_def456",
      "userId": "user_xyz789",
      "contactId": "contact_456",
      "contactName": "Mom",
      "messageCount": 5,
      "lastMessage": "How are you doing?",
      "lastMessageTime": 1725608000000,
      "createdAt": 1725604000000
    }
  ],
  "total": 2,
  "timestamp": 1725609600000
}
```

---

### 5. Get Contacts

**Endpoint**: `GET /api/contacts?userId=user_xyz789`

**Purpose**: List all contacts for user.

**Response** (Success - 200):
```json
{
  "contacts": [
    {
      "id": "contact_123",
      "name": "Sarah",
      "relationship": "close_friend",
      "characteristics": ["ambitious", "direct"],
      "interests": ["tech", "hiking"],
      "notes": "Software engineer",
      "updatedAt": 1725609600000
    },
    {
      "id": "contact_456",
      "name": "Mom",
      "relationship": "family",
      "characteristics": ["caring", "worried"],
      "interests": ["family", "cooking"],
      "notes": "Always asking how I'm doing",
      "updatedAt": 1725608000000
    }
  ],
  "total": 2,
  "timestamp": 1725609600000
}
```

---

### 6. Save About Me

**Endpoint**: `POST /api/about-me`

**Purpose**: Save or update user's communication style profile.

**Request**:
```json
{
  "userId": "user_xyz789",
  "communicationStyle": "casual, direct, authentic",
  "values": ["authenticity", "loyalty", "growth", "fun"],
  "preferredTone": "friendly",
  "notes": "I'm an introvert but love deep conversations"
}
```

**Response** (Success - 200):
```json
{
  "status": "saved",
  "aboutMe": {
    "userId": "user_xyz789",
    "communicationStyle": "casual, direct, authentic",
    "values": ["authenticity", "loyalty", "growth", "fun"],
    "preferredTone": "friendly",
    "notes": "I'm an introvert but love deep conversations",
    "updatedAt": 1725609600000
  }
}
```

---

### 7. Get Status

**Endpoint**: `GET /api/status`

**Purpose**: Health check. Used to detect if backend is available.

**Response** (Success - 200):
```json
{
  "status": "ok",
  "version": "1.0.0",
  "timestamp": 1725609600000,
  "agents": {
    "conversation": "ok",
    "learning": "ok",
    "context_manager": "ok",
    "risk_monitoring": "ok"
  }
}
```

---

## Error Response Format

**All errors return status 400-500 with**:
```json
{
  "error": "Human-readable error message",
  "errorCode": "ERROR_CODE",
  "statusCode": 400,
  "details": { "field": "error detail" },
  "requestId": "req_abc123",
  "timestamp": 1725609600000
}
```

---

## Common Error Codes

| Code | Meaning | Status |
|------|---------|--------|
| INVALID_REQUEST | Malformed JSON or missing fields | 400 |
| UNAUTHORIZED | Invalid/missing auth | 401 |
| FORBIDDEN | User doesn't have access | 403 |
| NOT_FOUND | Conversation/contact not found | 404 |
| RATE_LIMITED | Too many requests | 429 |
| BACKEND_UNAVAILABLE | Go backend down | 503 |
| INVALID_API_KEY | Provider API key invalid | 401 |
| MODEL_NOT_FOUND | Selected model doesn't exist | 400 |
| CONTEXT_MISSING | No About Me or Contact | 400 |
| TIMEOUT | Request took too long | 504 |

---

## Rate Limiting

- **Per user**: 100 requests/minute
- **Per conversation**: 10 requests/minute (for spam prevention)
- **Global**: 1000 requests/minute

**Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1725609660
```

---

## Request/Response Headers

**Request**:
```
Content-Type: application/json
Accept: application/json
User-Agent: Moly/1.0.0 (Chrome Extension)
```

**Response**:
```
Content-Type: application/json
X-Request-ID: req_abc123
X-Processing-Time: 1247ms
Cache-Control: no-cache, no-store
```

---

## Versioning

API version in path: `/api/v1/` (future)

Currently: `/api/` (v0 - unstable during development)

---

## Timeout Behavior

- **Default timeout**: 30 seconds
- **Connection timeout**: 5 seconds
- **Backend processing**: 1-3 seconds typical, up to 10 seconds on slow systems

**Handling**:
- If > 2 seconds: Show "Processing..." with spinner
- If > 5 seconds: Show elapsed time + "models can be slow on older systems"
- If > 30 seconds: Return timeout error, user can retry

---

## Data Types

```typescript
interface Suggestion {
  index: number;
  text: string;
  tone: "formal" | "friendly" | "dating";
  reasoning: string;
  confidence: number; // 0-1
}

interface Reflection {
  id: string;
  conversationId: string;
  characteristics: string[];
  interests: string[];
  communicationPreferences: string;
  intentions: string[];
  status: "pending_approval" | "approved" | "rejected";
  userQuotes?: string[];
  userEdits?: Partial<Reflection>;
  createdAt: number;
  approvedAt?: number;
}

interface SafetyCheckResult {
  alert_type: "crisis" | "illegal" | "none";
  severity: "immediate" | "high" | "warning";
  title: string;
  message: string;
  indicators: string[];
  resources: CrisisResource[];
  recommendations: string[];
}

interface RiskWarning {
  riskLevel: "immediate" | "high" | "medium" | "low" | "clear";
  pattern?: "manipulation" | "boundary" | "scam" | "harm" | "insincerity";
  severity: number; // 0-10
  educationalQuestions: string[];
  principles: CommunicationPrinciple[];
  alternatives: string[];
  recommendation: "proceed" | "educate_first" | "escalate";
  message: string;
}

interface Message {
  id: string;
  conversationId: string;
  role: "user" | "assistant";
  content: string;
  type: "message" | "question" | "suggestion";
  metadata?: Record<string, unknown>;
  timestamp: number;
}

interface Contact {
  id: string;
  userId: string;
  name: string;
  relationship: "close_friend" | "family" | "work" | "romantic" | "new";
  characteristics: string[];
  interests: string[];
  communicationPreferences: string;
  notes: string;
  reflections: Reflection[];
  createdAt: number;
  updatedAt: number;
}

interface Conversation {
  id: string;
  userId: string;
  contactId: string;
  contactName: string;
  messageCount: number;
  lastMessage: string;
  lastMessageTime: number;
  createdAt: number;
}
```

---

## Example Flow: Complete Conversation

```
1. Extension opens → GET /api/status → Check backend available

2. User creates conversation → Load contacts → GET /api/contacts

3. User selects Sarah → GET /api/context

4. User types message → POST /api/conversation/generate
   ├─ 2 seconds elapsed → Show "Processing..."
   ├─ 5 seconds elapsed → Show time + helpful message
   ├─ Response received → Show suggestions + reflection

5. User picks suggestion → POST /api/conversation/feedback
   ├─ Record choice
   ├─ Merge reflection to Sarah's profile
   ├─ Update learning agent

6. User creates new message with Mom → GET /api/context
   ├─ About Me loaded
   ├─ Mom's profile loaded (enriched from previous conversations)
   ├─ Richer context available → Better suggestions

7. All data persists → Next time: Even richer context
```

---

This API is designed to be:
- **Stateless** — Each request is self-contained
- **Predictable** — Consistent response formats
- **Resilient** — Graceful error handling
- **Transparent** — Request IDs for debugging
- **Safe** — No sensitive data in URLs
