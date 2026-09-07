package agents

// ContextManagerAgent - Intelligent knowledge base management
//
// Manages:
// - About Me: User's own communication profile
// - Contacts: User's observations of contacts (NOT surveillance)
// - Conversation History: Messages, questions, suggestions
// - Reflections: Extracted insights, pending approval
// - Intelligent Context Retrieval: Surfaces relevant context when needed
//
// Key principle:
// - Stores user's OBSERVATIONS of contacts, never analyzes contacts
// - When user says "Sarah is direct", that's stored
// - Never stores "Sarah responded in 2 hours" or analyzes Sarah's behavior
//
// See: /MOLY_V2_ARCHITECTURE/03_AGENT_PROMPTS.md - Context Manager section
// See: /MOLY_V2_ARCHITECTURE/06_DATABASE_SCHEMA.md
// Deadline: Sep 18-19, 2026

// TODO: Implement ContextManagerAgent interface and knowledge base
