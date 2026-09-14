# MOLY VISION & PRINCIPLES

## What is Moly?

Moly is an **AI coaching chatbot** that helps you craft better messages through natural conversation. Instead of forcing you through setup forms, Moly learns about you as you naturally chat, then provides increasingly intelligent suggestions tailored to your communication style and relationships.

## The Problem Moly Solves

Writing good messages is hard:
- You second-guess your tone and wording
- You forget important context about the person
- Different relationships need different approaches
- You want to improve how you communicate over time

**Moly's solution:** A thinking partner that asks clarifying questions as needed, remembers context from past conversations, and generates suggestions matched to how you actually communicate.

## Core Philosophy

### Natural Flow Over Forced Setup
- No mandatory profile creation
- Context gathered through conversation
- User chooses what to share, when
- System learns progressively

### Agentic, Not Prescriptive
- System asks clarifying questions, doesn't assume
- Agents extract and reason about information
- Suggestions are options, never commands
- User maintains full control and autonomy

### Privacy & Respect
- All data stored locally per user
- No surveillance or monitoring
- Context used only to improve suggestions
- User can opt-out of context retention

### Multi-Session Persistence
- Context survives across logout/login
- Incomplete clarifications resume automatically
- Relationship history is preserved
- Learning accumulates over time

## Architecture at a Glance

```
User sends message
       ↓
Agents extract facts, relationships, intent
       ↓
Database stores context + clarification questions
       ↓
API returns suggestions with full context
       ↓
Frontend displays + handles user responses
       ↓
User answers clarifications or sends new message
       ↓
[Loop continues, context accumulates]
```

## Key Features

### 1. Natural Conversation
- Chat naturally, no forms
- Moly asks what it needs to know
- Context builds organically

### 2. Intelligent Suggestions
- Generated with full context (AboutMe, Contacts, History)
- Matched to your communication style
- Multiple options to choose from
- Can be adapted before sending

### 3. Relationship Learning
- Saves what you tell Moly about people
- Tracks communication patterns
- Improves suggestions over time

### 4. Clarification Resumption
- If you log out mid-question, it resumes
- Doesn't force you to re-answer
- Works across days/weeks

### 5. Safety & Ethics
- Pattern detection for risky communication
- Educational guidance, not blocking
- Respects user autonomy
- Transparent about its reasoning

## Development Status

✅ **Complete:**
- Email/password authentication with multi-user isolation
- Agentic message processing pipeline
- Fact extraction and clarification question generation
- Database-backed context persistence
- API with proper authorization
- Chrome extension UI

🚀 **Ready for:**
- Multi-session context resumption testing
- User research and feedback
- Production deployment

## How to Contribute

See CONTRIBUTING.md for code guidelines and workflow.

For questions about architecture, see ARCHITECTURE.md.

For setup and development, see DEVELOPMENT.md.
