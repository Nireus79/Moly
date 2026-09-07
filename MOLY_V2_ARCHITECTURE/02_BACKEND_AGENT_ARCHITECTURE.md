# Moly Backend Agent Architecture

**Date**: September 6, 2026  
**Status**: Design Specification  
**Version**: 1.0

---

## Overview

Backend agents replace hardcoded flows with adaptive, reasoning-based decision making. Three specialized agents handle the complete conversation lifecycle:

1. **Conversation Agent** — orchestrates single conversation
2. **Learning Agent** — tracks patterns over time
3. **Context Manager Agent** — intelligent knowledge base

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│ EXTENSION (Browser)                                      │
├─────────────────────────────────────────────────────────┤
│ - Detects user action (sends message)                  │
│ - Sends to backend: { conversationId, message, mode }  │
│ - Receives: { questions, suggestions, reflection }    │
│ - Renders UI                                           │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP/JSON
                       ↓
┌─────────────────────────────────────────────────────────┐
│ BACKEND (Go + Claude API)                               │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌────────────────────────────────────────────────┐    │
│  │ API Gateway                                    │    │
│  │ POST /api/conversation/generate                │    │
│  │ - Routes to appropriate agent                  │    │
│  │ - Handles auth, rate limiting, logging         │    │
│  └────────────────────────────────────────────────┘    │
│                       ↓                                 │
│  ┌────────────────────────────────────────────────┐    │
│  │ CONVERSATION AGENT (Orchestrator)              │    │
│  │                                                │    │
│  │ Agentic Loop:                                  │    │
│  │ 1. Analyze: "What do we need?"                │    │
│  │ 2. Decide: Context gathering vs. suggestion   │    │
│  │ 3. Execute: Call tools or ask user            │    │
│  │ 4. Reflect: Learn from response               │    │
│  │ 5. Decide: Next step                          │    │
│  │                                                │    │
│  │ Tools Available:                               │    │
│  │ - contextManager.getRelevantContext()          │    │
│  │ - contextManager.saveReflection()              │    │
│  │ - learningAgent.getUserProfile()               │    │
│  │ - learningAgent.getContactProfile()            │    │
│  │ - generateSocraticQuestions()                  │    │
│  │ - generateSuggestions()                        │    │
│  │ - checkSafety()                                │    │
│  │ - evaluateConstitution()                       │    │
│  │ - extractContext()                             │    │
│  └────────────────────────────────────────────────┘    │
│                       ↓                                 │
│  ┌────────────────────────────────────────────────┐    │
│  │ LEARNING AGENT (Pattern Recognition)           │    │
│  │                                                │    │
│  │ Responsibilities:                              │    │
│  │ - Build user behavioral profile                │    │
│  │ - Track suggestion effectiveness               │    │
│  │ - Identify communication patterns              │    │
│  │ - Store (NO contact surveillance)              │    │
│  │                                                │    │
│  │ Tools Available:                               │    │
│  │ - database.getUserProfile(userId)              │    │
│  │ - database.updateUserProfile()                 │    │
│  │ - database.recordSuggestionChoice()            │    │
│  │ - database.getPatterns()                       │    │
│  └────────────────────────────────────────────────┘    │
│                       ↓                                 │
│  ┌────────────────────────────────────────────────┐    │
│  │ CONTEXT MANAGER AGENT (Knowledge Base)         │    │
│  │                                                │    │
│  │ Responsibilities:                              │    │
│  │ - Store/retrieve About Me                      │    │
│  │ - Store/retrieve Contact notes                 │    │
│  │ - Manage conversation history                  │    │
│  │ - Surface relevant context intelligently       │    │
│  │                                                │    │
│  │ Tools Available:                               │    │
│  │ - database.get/setAboutMe()                    │    │
│  │ - database.get/setContactProfile()             │    │
│  │ - database.appendMessage()                     │    │
│  │ - database.getRelevantHistory()                │    │
│  └────────────────────────────────────────────────┘    │
│                       ↓                                 │
│  ┌────────────────────────────────────────────────┐    │
│  │ SHARED TOOLS LAYER                             │    │
│  │                                                │    │
│  │ - SafetyChecker                                │    │
│  │ - ConstitutionEvaluator                        │    │
│  │ - ContextExtractor                             │    │
│  │ - SuggestionGenerator                          │    │
│  │ - QuestionGenerator                            │    │
│  │ - LLMClient (calls Claude)                     │    │
│  └────────────────────────────────────────────────┘    │
│                       ↓                                 │
│  ┌────────────────────────────────────────────────┐    │
│  │ DATA LAYER                                     │    │
│  │                                                │    │
│  │ Database Tables:                               │    │
│  │ - users (id, about_me, preferences)            │    │
│  │ - contacts (userId, contactId, name, notes)    │    │
│  │ - conversations (id, userId, contactId)        │    │
│  │ - messages (id, conversationId, role, content) │    │
│  │ - user_profiles (userId, patterns, choices)    │    │
│  │ - reflections (id, conversationId, insights)   │    │
│  └────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

---

## Agent 1: Conversation Agent (Orchestrator)

### Responsibility
Runs the complete 5-phase interaction for one user message/request.

### Decision Tree

```
User sends message to Moly
        ↓
Conversation Agent starts agentic loop
        ↓
DECISION 1: What's happening?
├─ "Tell me about Sarah" → Context gathering phase
├─ "I don't know what to say" → Generate questions phase
├─ [Message from Sarah] → Full pipeline (context + safety + suggestions)
└─ "Make this more casual" → Refinement phase

        ↓
DECISION 2: What context do we have?
├─ Call: contextManager.getRelevantContext(conversationId)
├─ Get: { aboutMe, contactProfile, conversationHistory }
└─ If missing: Ask user OR inform user to build it

        ↓
DECISION 3: Safety first?
├─ Call: checkSafety(message)
├─ If crisis/illegal: Return crisis resources + stop
└─ If ok: Continue

        ↓
DECISION 4: Generate what?
├─ If context gathering: Generate Socratic questions
│   └─ Call: generateSocraticQuestions(contactName, context, conversationHistory)
├─ If full pipeline: Do all:
│   ├─ Call: evaluateConstitution(message) in background
│   ├─ Call: generateSuggestions(aboutMe, contactProfile, intention, history)
│   └─ Extract insights for reflection
└─ If refinement: Call: generateSuggestions(previousSuggestion, refinementRequest)

        ↓
DECISION 5: What to return?
├─ Return: { questions, suggestions, reflection, safetyAlert, constitutionConcerns }
└─ Let extension decide what to display
```

### Agent Loop (Pseudo-code)

```typescript
async function conversationAgentLoop(input: {
  conversationId: string;
  userId: string;
  userMessage: string;
  mode: 'socratic' | 'direct';
  tone: 'formal' | 'friendly' | 'dating';
}): Promise<ConversationResponse> {
  
  // Initialize agent state
  const state = {
    analyzed: false,
    contextGathered: false,
    safetyChecked: false,
    suggestionsGenerated: false,
    reflectionExtracted: false,
    errors: []
  };

  try {
    // PHASE 1: ANALYSIS
    const analysis = await agent.think(
      `User message: "${input.userMessage}"
       Conversation context: [${context}]
       What phase are we in? What decisions need to be made?`
    );
    state.analyzed = true;

    // PHASE 2: CONTEXT GATHERING
    const relevantContext = await contextManager.getRelevantContext(
      input.conversationId,
      input.userId
    );
    state.contextGathered = true;

    // If missing critical context, ask for it first
    if (!relevantContext.aboutMe || !relevantContext.contactProfile) {
      const questions = await generateContextQuestions(
        relevantContext,
        input.userMessage
      );
      return {
        phase: 'context_gathering',
        questions,
        suggestions: [],
        message: 'Let me learn more about you first...'
      };
    }

    // PHASE 3: SAFETY CHECK
    const safetyResult = await safetyChecker.check(input.userMessage);
    state.safetyChecked = true;

    if (safetyResult.alert_type !== 'none') {
      return {
        phase: 'safety_alert',
        safety: safetyResult,
        suggestions: [],
        questions: []
      };
    }

    // PHASE 4: GENERATE SUGGESTIONS
    const suggestions = await generateSuggestions({
      aboutMe: relevantContext.aboutMe,
      contactProfile: relevantContext.contactProfile,
      intention: analysis.intention,
      conversationHistory: relevantContext.conversationHistory,
      userBehaviorProfile: await learningAgent.getUserProfile(input.userId),
      mode: input.mode,
      tone: input.tone,
      userMessage: input.userMessage
    });
    state.suggestionsGenerated = true;

    // PHASE 4B: ETHICS CHECK (background, doesn't block)
    const constitutionCheck = await evaluateConstitution(input.userMessage);
    if (constitutionCheck.violations.length > 0) {
      // Generate Socratic follow-up questions
      const educationalQuestions = await generateEducationalQuestions(
        constitutionCheck,
        input.userMessage
      );
      return {
        ...suggestions,
        constitutionConcerns: constitutionCheck,
        educationalQuestions
      };
    }

    // PHASE 5: REFLECTION & LEARNING
    const reflection = await extractContext(
      input.userMessage,
      relevantContext,
      suggestions
    );
    state.reflectionExtracted = true;

    // Store reflection for user approval
    await contextManager.saveReflection(
      input.conversationId,
      reflection,
      'pending_approval'
    );

    // Record in learning agent
    await learningAgent.recordInteraction({
      userId: input.userId,
      conversationId: input.conversationId,
      userMessage: input.userMessage,
      suggestionsGenerated: suggestions.suggestions.length,
      chosenIndex: -1, // Will be updated when user picks one
      success: null // Will be updated based on contact response
    });

    return {
      phase: 'suggestions_ready',
      suggestions: suggestions.suggestions,
      reflection: reflection,
      questions: [], // Optional Socratic follow-ups
      constitutionConcerns: []
    };

  } catch (error) {
    return {
      phase: 'error',
      error: error.message,
      suggestions: []
    };
  }
}
```

### Key Design Decisions

1. **Agentic Loop** — Agent thinks before acting, decides what to do, executes decision
2. **Tool Calling** — Agent calls tools (context manager, safety checker, etc.) dynamically
3. **Graceful Degradation** — If one phase fails, still return something useful
4. **Background Tasks** — Ethics checks don't block suggestions (run in background)
5. **No Hardcoding** — Every response is generated by reasoning, not templates

---

## Agent 2: Learning Agent (Pattern Recognition)

### Responsibility
Build and maintain user behavioral profile. Never monitors contacts.

### What It Learns

**About User (Stored):**
```
{
  userId: "user123",
  communicationProfile: {
    avgToneWords: ["authentic", "direct", "warm"],
    formalityLevel: 0.3, // 0 = very casual, 1 = very formal
    emojiUsage: 0.2, // frequency
    messageLengthAvg: 120, // characters
    usesHumor: true,
    usesEmphasis: ["!!", "really", "so"],
    formattingPreference: "paragraph" // vs bullet, vs short
  },
  communicationGoals: {
    opening: 45, // count of opening messages user crafted
    deepening: 120, // deepening connection messages
    apology: 15,
    clarification: 42,
    celebration: 87
  },
  suggestionChoices: {
    totalGenerated: 300,
    totalChosen: 195,
    avgChoiceRate: 0.65,
    preferred: {
      formal: 0.1,
      friendly: 0.65,
      dating: 0.25
    },
    frequentEdits: ["make more casual", "shorten", "add emoji"]
  },
  successMetrics: {
    positive_response_rate: 0.72, // inferred from user feedback
    conversation_continuation_rate: 0.81,
    contact_reengagement: true,
    user_confidence_trend: "increasing"
  },
  emergingPersonality: [
    "values authenticity",
    "prefers casual communication",
    "celebrates others",
    "avoids conflict initially",
    "learns from patterns quickly"
  ],
  growthTrajectory: {
    joinDate: "2026-01-15",
    totalInteractions: 450,
    uniqueContacts: 18,
    averageContactFrequency: 25 // interactions per contact
  }
}
```

**About Contacts (User's Observations ONLY):**
```
{
  contactId: "sarah123",
  userId: "user123",
  name: "Sarah",
  notes: "Software engineer, career-focused, direct communicator, loves hiking",
  characteristicsTags: [
    "ambitious",
    "straightforward",
    "outdoor-oriented",
    "technical",
    "loyal"
  ],
  communicationPreferencesObserved: "Doesn't like small talk, appreciates direct feedback",
  relationshipPhase: "deepening_connection",
  interactionCount: 25,
  lastInteraction: "2026-09-04"
  // NOTE: NO monitoring of how Sarah responds
  // NO tracking of her message patterns
  // NO behavior analysis of the contact
}
```

### Learning Agent API

```typescript
interface LearningAgent {
  // Get profile
  getUserProfile(userId: string): Promise<UserBehavioralProfile>;
  getContactProfile(userId: string, contactId: string): Promise<ContactObservations>;
  
  // Update profile
  recordInteraction(data: {
    userId: string;
    conversationId: string;
    userMessage: string;
    suggestionsGenerated: number;
    chosenSuggestion?: string;
    chosenTone?: string;
  }): Promise<void>;
  
  recordSuggestionChoice(data: {
    userId: string;
    conversationId: string;
    suggestionIndex: number;
    modifiedText?: string;
    requestedModification?: string;
  }): Promise<void>;
  
  recordSuccess(data: {
    userId: string;
    contactId: string;
    conversationId: string;
    contactResponse?: string;
    userFeedback?: "positive" | "neutral" | "negative";
  }): Promise<void>;
  
  // Analytics
  getPatterns(userId: string): Promise<UserPatterns>;
  getGrowthTrajectory(userId: string): Promise<GrowthData>;
  
  // Inference
  predictPreferredTone(userId: string): Promise<"formal" | "friendly" | "dating">;
  detectEmergingValues(userId: string): Promise<string[]>;
}
```

### Learning Loop Example

```
User interaction 1:
├─ Message: "Sarah got promoted!"
├─ Chosen suggestion: Friendly tone
├─ Modification: Made it shorter
└─ Learning Agent records: user prefers friendly, edits for brevity

User interaction 2:
├─ Message: "Mom is upset with me"
├─ Suggestion generated: Formal tone
├─ Learning Agent: "User avoided formal last time, predict friendly?"
└─ Offers friendly as first option

User interaction 50:
├─ Accumulated data: user consistently chooses friendly, casual
├─ Emerging pattern: values authenticity over perfection
├─ Conversation Agent uses this: "This user would prefer casual but genuine"

User interaction 100:
├─ Behavior profile enriched
├─ Can predict: user's tone preference, edit preferences, communication goals
├─ Can detect: "User seems more confident now" → adjust suggestions
└─ Learning Agent informs all future suggestions
```

---

## Agent 3: Context Manager Agent (Knowledge Base)

### Responsibility
Intelligent storage and retrieval of user knowledge (About Me, Contacts, History).

### Data Model

```
About Me (Single per user):
├─ communicationStyle: "casual, direct, authentic"
├─ values: ["authenticity", "loyalty", "growth", "fun"]
├─ preferredTone: "friendly"
├─ notes: "Free-form text user provides"
└─ lastUpdated: timestamp

Contacts (Many per user):
├─ name: "Sarah"
├─ relationship: "close friend" | "family" | "work" | "romantic" | "new"
├─ characteristics: ["ambitious", "direct", "loves hiking"]
├─ interests: ["tech", "hiking", "coffee"]
├─ communicationPreferences: "No small talk, appreciates directness"
├─ notes: User's observations + reflected insights
└─ reflections: [ { timestamp, insights, userApproved } ]

Conversation History:
├─ messages: [
│  { role: "user", content, timestamp, metadata: {} },
│  { role: "assistant", content, timestamp, type: "question" | "suggestion" },
│  { role: "user", content: "Make it more casual", timestamp }
│ ]
└─ context: { contactId, userId, createdAt, lastActivity }
```

### Context Manager API

```typescript
interface ContextManager {
  // About Me
  getAboutMe(userId: string): Promise<AboutMe>;
  setAboutMe(userId: string, aboutMe: AboutMe): Promise<void>;
  updateAboutMe(userId: string, updates: Partial<AboutMe>): Promise<void>;
  
  // Contacts
  getContact(userId: string, contactId: string): Promise<Contact>;
  getContacts(userId: string): Promise<Contact[]>;
  createContact(userId: string, contact: Contact): Promise<Contact>;
  updateContact(userId: string, contactId: string, updates: Partial<Contact>): Promise<void>;
  appendContactNotes(userId: string, contactId: string, notes: string): Promise<void>;
  
  // Conversations
  getConversation(conversationId: string): Promise<Conversation>;
  appendMessage(conversationId: string, message: Message): Promise<void>;
  
  // Smart retrieval
  getRelevantContext(conversationId: string, userId: string): Promise<{
    aboutMe: AboutMe;
    contactProfile: Contact;
    conversationHistory: Message[];
    userBehaviorProfile: UserBehavioralProfile;
  }>;
  
  // Reflections
  saveReflection(conversationId: string, reflection: Reflection, status: "pending" | "approved"): Promise<void>;
  getReflectionForApproval(conversationId: string): Promise<Reflection>;
  approveReflection(conversationId: string, reflection: Reflection, edits?: Partial<Reflection>): Promise<void>;
  mergeReflectionToContact(userId: string, contactId: string, reflection: Reflection): Promise<void>;
}
```

### Intelligent Retrieval

```
Query: Get relevant context for a conversation

Input: 
├─ conversationId
├─ userId
└─ currentUserMessage

Process:
1. Load conversation history (last N messages, full context)
2. Identify contact from conversation
3. Load About Me
4. Load Contact profile
5. Load user behavioral profile
6. Rank by relevance: recent > frequently mentioned > high confidence
7. Trim to context length limits (stay under token limit)

Return: { aboutMe, contactProfile, conversationHistory, userBehaviorProfile }
```

---

## Shared Tools (All Agents Use These)

### 1. SafetyChecker
```typescript
interface SafetyChecker {
  check(message: string): Promise<{
    alert_type: 'crisis' | 'illegal' | 'none';
    severity: 'immediate' | 'high' | 'warning';
    title: string;
    message: string;
    indicators: string[];
    resources: CrisisResource[];
    recommendations: string[];
  }>;
}
```
**Responsibility**: Detect crisis language, illegal intent, immediate harm

### 2. ConstitutionEvaluator
```typescript
interface ConstitutionEvaluator {
  evaluate(message: string): Promise<{
    violations: ConstitutionViolation[];
    aligned_principles: string[];
    overall_risk_level: 'low' | 'medium' | 'high';
    recommendations: string[];
    is_constitutional: boolean;
  }>;
}
```
**Responsibility**: Evaluate against ethical principles, educational (not blocking)

### 3. ContextExtractor
```typescript
interface ContextExtractor {
  extract(
    userMessage: string,
    context: Context,
    suggestions: Suggestion[]
  ): Promise<{
    characteristics: string[];
    interests: string[];
    communicationPreferences: string[];
    relationshipPhase: string;
    intentions: string[];
  }>;
}
```
**Responsibility**: Extract insights from conversation for reflection

### 4. SuggestionGenerator
```typescript
interface SuggestionGenerator {
  generate(input: {
    aboutMe: AboutMe;
    contactProfile: Contact;
    conversationHistory: Message[];
    userMessage: string;
    intention: string;
    userBehaviorProfile: UserBehavioralProfile;
    mode: 'socratic' | 'direct';
    tone: 'formal' | 'friendly' | 'dating';
  }): Promise<{
    suggestions: {
      text: string;
      tone: string;
      reasoning: string;
      confidence: number;
    }[];
  }>;
}
```
**Responsibility**: Generate 3-5 personalized suggestions

### 5. QuestionGenerator
```typescript
interface QuestionGenerator {
  generateSocratic(input: {
    contactName: string;
    conversationHistory: Message[];
    intention?: string;
    aboutMe: AboutMe;
    userBehaviorProfile: UserBehavioralProfile;
  }): Promise<{
    questions: string[];
    reasoning: string;
  }>;
  
  generateEducational(input: {
    constitutionViolations: ConstitutionViolation[];
    userMessage: string;
  }): Promise<{
    questions: string[];
    principles: CommunicationPrinciple[];
  }>;
  
  generateContextGathering(input: {
    missingContext: string[]; // ["aboutMe", "contactProfile", "intention"]
    userMessage: string;
  }): Promise<{
    questions: string[];
  }>;
}
```
**Responsibility**: Generate questions that feel conversational, not templated

### 6. LLMClient (Claude API)
```typescript
interface LLMClient {
  think(prompt: string): Promise<{
    reasoning: string;
    conclusion: string;
  }>;
  
  generate(prompt: string, maxTokens?: number): Promise<string>;
  
  agentic(tools: Tool[], systemPrompt: string, messages: Message[]): Promise<{
    thinking: string;
    action: {
      type: "tool_use" | "text_response";
      name?: string;
      input?: any;
      text?: string;
    };
  }>;
}
```
**Responsibility**: Call Claude API with appropriate prompts

---

## Communication Protocol (Extension ↔ Backend)

### Request Format

```json
POST /api/conversation/generate

{
  "conversationId": "conv_abc123",
  "userId": "user_xyz789",
  "userMessage": "Sarah just got promoted and I don't know what to say",
  "mode": "socratic",
  "tone": "friendly",
  "type": "generate",
  "metadata": {
    "previousSuggestionChosen": 0,
    "userModified": "Made it shorter",
    "contactResponse": "That's so sweet, thanks!",
    "userFeedback": "positive"
  }
}
```

### Response Format

```json
{
  "phase": "suggestions_ready",
  "suggestions": [
    {
      "text": "That's amazing! I'm so proud of you. Let's celebrate soon!",
      "tone": "friendly",
      "reasoning": "Celebrates achievement, genuine, matches your authentic style",
      "confidence": 0.92
    },
    ...
  ],
  "questions": [
    "What would celebrating look like to you two?",
    "Is she taking time off to enjoy this milestone?"
  ],
  "reflection": {
    "characteristics": ["ambitious", "achievement-focused"],
    "interests": ["career growth"],
    "communicationPreferences": "Direct, celebrates achievements",
    "userIntention": "Express genuine pride",
    "status": "pending_approval"
  },
  "constitutionConcerns": null,
  "safety": null,
  "processingTimeMs": 1247
}
```

### Error Response

```json
{
  "phase": "error",
  "error": "Backend unavailable",
  "fallback": {
    "suggestions": "Offline mode - use cached suggestions",
    "questions": "Tell me more about Sarah"
  }
}
```

---

## Workflow: Complete Flow Example

```
SCENARIO: User says "Sarah got promoted, I want to celebrate with her"

1. EXTENSION:
   POST /api/conversation/generate
   {
     conversationId: "conv_123",
     userId: "user_456",
     userMessage: "Sarah got promoted, I want to celebrate with her",
     mode: "direct",
     tone: "friendly"
   }

2. BACKEND - CONVERSATION AGENT starts:
   
   a) ANALYZE:
      Agent: "User wants to celebrate Sarah's promotion.
              This is full pipeline: context gathering (if needed) + 
              safety check + suggestion generation + reflection"
   
   b) CONTEXT:
      Agent calls: contextManager.getRelevantContext(conv_123, user_456)
      Returns: {
        aboutMe: { style: "casual, authentic", values: [...] },
        contactProfile: { name: "Sarah", characteristics: [...] },
        conversationHistory: [...last 10 messages...],
        userBehaviorProfile: { prefersTone: "friendly", ... }
      }
   
   c) SAFETY:
      Agent calls: safetyChecker.check("Sarah got promoted...")
      Returns: { alert_type: "none", ... }
   
   d) INTENTION DETECTION:
      Agent: "User's intention: celebrate achievement, express pride"
   
   e) SUGGESTION GENERATION:
      Agent calls: suggestionGenerator.generate({
        aboutMe, contactProfile, conversationHistory,
        userMessage: "...",
        intention: "celebrate achievement",
        userBehaviorProfile,
        mode: "direct",
        tone: "friendly"
      })
      Returns: 3 suggestions with reasoning
   
   f) ETHICS (background):
      Agent calls: constitutionEvaluator.evaluate("...")
      Result: aligned with "authentic celebration" principle
   
   g) REFLECTION:
      Agent calls: contextExtractor.extract(...)
      Gets: {
        characteristics: ["ambitious", "achievement-focused"],
        communicationPreferences: "Direct but warm",
        intention: "celebrate genuinely"
      }
      Agent calls: contextManager.saveReflection(conv_123, reflection)
   
   h) LEARNING:
      Agent calls: learningAgent.recordInteraction({
        userId, conversationId, userMessage,
        suggestionsGenerated: 3
      })

3. BACKEND returns:
   {
     "phase": "suggestions_ready",
     "suggestions": [
       { text: "...", tone: "friendly", confidence: 0.92 },
       { text: "...", tone: "friendly", confidence: 0.88 },
       { text: "...", tone: "friendly", confidence: 0.85 }
     ],
     "reflection": { ... pending_approval ... }
   }

4. EXTENSION:
   - Displays 3 suggestions
   - Shows reflection modal
   - User picks suggestion #1
   - User approves reflection (or edits)

5. EXTENSION sends feedback:
   POST /api/conversation/feedback
   {
     conversationId: "conv_123",
     userId: "user_456",
     suggestionChosen: 0,
     reflectionApproved: true,
     reflectionEdits: {}
   }

6. BACKEND - LEARNING AGENT:
   Agent calls: learningAgent.recordSuggestionChoice({
     userId, conversationId, suggestionIndex: 0
   })
   Agent calls: contextManager.approveReflection(conv_123, reflection)
   Agent calls: contextManager.mergeReflectionToContact(
     user_456, "sarah_123", reflection
   )

7. DONE: Next time user chats about Sarah, agent has richer context
```

---

## Implementation Considerations

### 1. Token Management
- Conversation Agent must stay within token limits
- Context Manager trims history intelligently
- Solution: Track token usage, implement smart trimming

### 2. Latency
- Each agent step = backend call + Claude API call
- Typical response time: 1-3 seconds on average
- Solution: Background tasks (ethics checks), streaming where possible

### 3. Cost
- Agent architecture = more API calls
- Typical cost: ~3-5 Claude calls per user message
- Solution: Cache profiles, batch queries, implement cost tracking

### 4. Error Handling
- If backend down: Return cached suggestions + simple fallback questions
- If Claude API error: Use fallback templates temporarily
- Solution: Graceful degradation at each layer

### 5. Privacy
- No contact surveillance data ever stored
- User behavioral data separate from contact data
- Solution: Strict database schema, clear table separation, audit logging

### 6. Scaling
- Single agent instance: handles ~100 concurrent conversations
- Solution: Agent pool/queue, database as state store, horizontal scaling

---

## Database Schema (Go/PostgreSQL)

```sql
-- Users table
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR UNIQUE,
  created_at TIMESTAMP,
  about_me JSONB,
  preferences JSONB
);

-- Contacts table
CREATE TABLE contacts (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  name VARCHAR,
  relationship VARCHAR,
  characteristics JSONB,
  notes TEXT,
  reflections JSONB[],
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- Conversations table
CREATE TABLE conversations (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  contact_id UUID REFERENCES contacts(id),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

-- Messages table
CREATE TABLE messages (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations(id),
  role VARCHAR, -- 'user' or 'assistant'
  content TEXT,
  type VARCHAR, -- 'message', 'question', 'suggestion'
  metadata JSONB,
  created_at TIMESTAMP
);

-- User behavioral profiles
CREATE TABLE user_profiles (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id) UNIQUE,
  communication_profile JSONB,
  communication_goals JSONB,
  suggestion_choices JSONB,
  success_metrics JSONB,
  emerging_personality JSONB,
  growth_trajectory JSONB,
  updated_at TIMESTAMP
);

-- Reflections (pending approval)
CREATE TABLE reflections (
  id UUID PRIMARY KEY,
  conversation_id UUID REFERENCES conversations(id),
  insights JSONB,
  status VARCHAR, -- 'pending', 'approved', 'rejected'
  user_edits JSONB,
  created_at TIMESTAMP,
  approved_at TIMESTAMP
);
```

---

## Key Design Principles

1. **No Hardcoding** — Every response generated by reasoning, not templates
2. **Agent Autonomy** — Agents decide what to do, not follow scripts
3. **User Privacy** — Contact data never analyzed, only stored
4. **Graceful Degradation** — Works offline, works when API slow, doesn't crash
5. **Learning Over Time** — Improves with each interaction
6. **Transparency** — User sees reasoning, can approve/edit
7. **Modularity** — Agents can work independently or together
8. **Stateful** — Full conversation history available for context

---

## Next Steps

1. **Implement Conversation Agent** — The orchestrator
2. **Implement Tools** — Safety, Constitution, Context, Suggestion generators
3. **Implement Data Layer** — Database schema, queries
4. **Implement Learning Agent** — User profile tracking
5. **Implement Context Manager** — Storage and retrieval
6. **Integration Testing** — Full flow end-to-end
7. **Performance Tuning** — Latency, cost, accuracy

---

This architecture replaces scripts with reasoning. Every interaction flows through agentic decision-making, ensuring authenticity and avoiding the template trap.

