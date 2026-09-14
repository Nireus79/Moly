# MOLY API REFERENCE

Base URL: `http://localhost:8080`

All requests (except auth) require:
```
Authorization: Bearer <session-token>
Content-Type: application/json
```

## Authentication

### Register

```
POST /api/auth/register
```

**Request:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "secure_password"
}
```

**Response (200):**
```json
{
  "success": true,
  "sessionToken": "sess_abc123...",
  "userId": "user_123",
  "expiresAt": 1694097600
}
```

**Errors:**
- 400: Email already exists / Invalid input
- 500: Server error

---

### Login

```
POST /api/auth/login
```

**Request:**
```json
{
  "email": "john@example.com",
  "password": "secure_password"
}
```

**Response (200):**
```json
{
  "success": true,
  "sessionToken": "sess_abc123...",
  "userId": "user_123",
  "expiresAt": 1694097600
}
```

**Errors:**
- 401: Invalid credentials
- 404: User not found
- 500: Server error

---

### Logout

```
POST /api/auth/logout
```

**Headers:** Requires valid token

**Response (200):**
```json
{
  "success": true
}
```

---

## Message Processing

### Process Message (Phase 5)

```
POST /api/v2/message-processor
```

**Request:**
```json
{
  "message": "I need to talk to my boss about the project delay",
  "conversationId": "conv_123",
  "aboutMe": {
    "communicationStyle": "direct",
    "coreValues": ["honesty", "clarity"],
    "tonePreference": "professional"
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "phase1": {
    "facts": [
      {
        "id": "fact_1",
        "type": "subject",
        "value": "boss",
        "evidence": "I need to talk to my boss",
        "confidence": 0.95
      }
    ],
    "shifts": []
  },
  "phase2": {
    "clarifications": []
  },
  "phase3": {
    "unknown_contacts": [],
    "created_contacts": ["contact_boss"]
  },
  "phase4": {
    "saved_attributes": []
  },
  "action_required": {
    "needsClarification": true,
    "clarificationQs": [
      {
        "id": "q_123",
        "type": "subject_clarification",
        "question": "Who is your boss?",
        "context": "You mentioned 'my boss'",
        "options": ["Manager", "Director", "CEO", "Other"],
        "priority": 1,
        "linkedFacts": ["fact_1"],
        "status": "pending",
        "createdAt": 1694097600
      }
    ],
    "temporaryFacts": [
      {
        "id": "temp_1",
        "type": "subject",
        "value": "boss",
        "linkedQuestionIds": ["q_123"]
      }
    ]
  }
}
```

**Errors:**
- 400: Invalid request
- 401: Unauthorized (invalid token)
- 500: Server error

---

## Clarifications

### Respond to Clarification Question

```
POST /api/v2/clarification/respond
```

**Request:**
```json
{
  "questionId": "q_123",
  "userResponse": "",
  "selectedOption": "Manager",
  "factId": "fact_1"
}
```

**Response (200):**
```json
{
  "success": true,
  "status": "clarification_processed",
  "nextQuestion": {
    "id": "q_124",
    "type": "context_clarification",
    "question": "Is this about a recent event?",
    "context": "Understanding timeline helps",
    "options": ["Yes, recent", "No, ongoing"],
    "priority": 2,
    "linkedFacts": ["fact_1"],
    "status": "pending",
    "createdAt": 1694097600
  },
  "factUpdated": {
    "id": "fact_1",
    "value": "boss_manager",
    "confidence": 0.98
  }
}
```

**Errors:**
- 400: Invalid question ID
- 401: Unauthorized
- 404: Question not found
- 500: Server error

### Get Pending Clarifications

```
GET /api/v2/clarification/pending
```

**Response (200):**
```json
{
  "success": true,
  "clarifications": [
    {
      "id": "q_123",
      "type": "subject_clarification",
      "question": "Who is your boss?",
      "context": "You mentioned 'my boss'",
      "options": ["Manager", "Director", "CEO", "Other"],
      "priority": 1,
      "linkedFacts": ["fact_1"],
      "status": "pending",
      "createdAt": 1694097600
    }
  ]
}
```

---

## About Me (User Profile)

### Get About Me

```
GET /api/v2/about-me
```

**Response (200):**
```json
{
  "success": true,
  "aboutMe": {
    "communicationStyle": "direct",
    "coreValues": ["honesty", "clarity"],
    "tonePreference": "professional",
    "preferences": "I prefer written over verbal"
  }
}
```

**Errors:**
- 401: Unauthorized
- 404: Profile not found (user is new)
- 500: Server error

### Set About Me

```
POST /api/v2/about-me
```

**Request:**
```json
{
  "communicationStyle": "direct",
  "coreValues": ["honesty", "clarity"],
  "tonePreference": "professional",
  "preferences": "I prefer written over verbal"
}
```

**Response (200):**
```json
{
  "success": true,
  "aboutMe": {
    "communicationStyle": "direct",
    "coreValues": ["honesty", "clarity"],
    "tonePreference": "professional",
    "preferences": "I prefer written over verbal"
  }
}
```

**Errors:**
- 400: Invalid input
- 401: Unauthorized
- 500: Server error

---

## Contacts

### List Contacts

```
GET /api/v2/contacts
```

**Response (200):**
```json
{
  "success": true,
  "contacts": [
    {
      "id": "contact_boss",
      "name": "Jane Smith",
      "relationship": "manager",
      "notes": "Technical PM, prefers data-driven decisions",
      "createdAt": 1694097600
    }
  ]
}
```

### Get Contact

```
GET /api/v2/contacts/:id
```

**Response (200):**
```json
{
  "success": true,
  "contact": {
    "id": "contact_boss",
    "name": "Jane Smith",
    "relationship": "manager",
    "notes": "Technical PM, prefers data-driven decisions",
    "createdAt": 1694097600
  }
}
```

### Create Contact

```
POST /api/v2/contacts
```

**Request:**
```json
{
  "name": "Jane Smith",
  "relationship": "manager",
  "notes": "Technical PM"
}
```

**Response (200):**
```json
{
  "success": true,
  "contact": {
    "id": "contact_abc123",
    "name": "Jane Smith",
    "relationship": "manager",
    "notes": "Technical PM",
    "createdAt": 1694097600
  }
}
```

### Update Contact

```
PUT /api/v2/contacts/:id
```

**Request:**
```json
{
  "name": "Jane Smith",
  "relationship": "manager",
  "notes": "Technical PM, prefers data-driven decisions"
}
```

**Response (200):**
```json
{
  "success": true,
  "contact": {
    "id": "contact_abc123",
    "name": "Jane Smith",
    "relationship": "manager",
    "notes": "Technical PM, prefers data-driven decisions",
    "createdAt": 1694097600
  }
}
```

### Delete Contact

```
DELETE /api/v2/contacts/:id
```

**Response (200):**
```json
{
  "success": true
}
```

---

## Conversations

### List Conversations

```
GET /api/v2/conversations
```

**Response (200):**
```json
{
  "success": true,
  "conversations": [
    {
      "id": "conv_123",
      "name": "Project Delay Discussion",
      "type": "work",
      "description": "Discussing the timeline impact",
      "createdAt": 1694097600
    }
  ]
}
```

### Create Conversation

```
POST /api/v2/conversations
```

**Request:**
```json
{
  "name": "Project Delay Discussion",
  "type": "work",
  "description": "Discussing the timeline impact"
}
```

**Response (200):**
```json
{
  "success": true,
  "conversation": {
    "id": "conv_123",
    "name": "Project Delay Discussion",
    "type": "work",
    "description": "Discussing the timeline impact",
    "createdAt": 1694097600
  }
}
```

---

## Response Format

All successful responses:
```json
{
  "success": true,
  "<resource>": { /* data */ }
}
```

All error responses:
```json
{
  "success": false,
  "error": "error message"
}
```

---

## HTTP Status Codes

- **200**: Success
- **400**: Bad request (invalid input, validation failed)
- **401**: Unauthorized (invalid/expired token)
- **403**: Forbidden (user doesn't own resource)
- **404**: Not found
- **500**: Server error

---

## Rate Limiting

Currently unlimited. Future: 100 requests per minute per user.

---

## See Also

- ARCHITECTURE.md: How endpoints connect to agents and database
- DEVELOPMENT.md: How to test endpoints locally
