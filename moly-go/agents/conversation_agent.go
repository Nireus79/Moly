package agents

// ConversationAgent - Orchestrates the 5-phase conversation flow
//
// Phases:
// 1. ANALYZE: Determine what type of interaction this is
// 2. CONTEXT: Gather relevant context (About Me, Contact, History)
// 3. SAFETY: Check for crisis/illegal content
// 4. RISK: Detect concerning user patterns
// 5. INTENTION: Understand user's actual goal
// 6. GENERATE: Create suggestions or questions
// 7. REFLECT: Extract insights for learning
//
// See: /MOLY_V2_ARCHITECTURE/03_AGENT_PROMPTS.md - Conversation Agent section
// See: /MOLY_V2_ARCHITECTURE/02_BACKEND_AGENT_ARCHITECTURE.md
// Deadline: Sep 14-16, 2026

// TODO: Implement ConversationAgent interface and decision tree
