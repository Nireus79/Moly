# Feature inventory (2026-10-09)

Built from the code: server routes in `moly-go/main.go`, handler bodies (tables read and written), and extension imports in `moly-extension/src`.
Decision column: **keep** (must work after the rebuild), **rebuild** (keep the purpose, new implementation), **drop** (needs your decision), **?** (unclear).

## 1. Server routes

| Route | Handler | Writes | Reads | Used by (extension) | Decision |
|---|---|---|---|---|---|
| POST /api/auth/register | RegisterHandler | users, about_me, system_context, sessions | — | backendConfig | keep |
| POST /api/auth/login | LoginHandler | sessions | users | backendConfig | keep |
| GET /api/auth/verify | VerifyTokenHandler | — | sessions, users | backendConfig | keep |
| POST /api/auth/logout | LogoutHandler | sessions | — | backendConfig | keep |
| POST /api/message-processor | MessageProcessorHandler | about_me, chat_messages, contacts, context_attributes, conversation_execution_state, conversations, reflections, safety_incidents, system_context | clarification_questions | agentClient, clarificationAPI | keep (the core) |
| POST /api/clarification/respond | ClarificationResponseHandler | about_me, clarification_questions | — | clarificationAPI, backendConfig | keep |
| POST /api/incoming-message/analyze | AnalyzeIncomingMessageHandler | — | about_me, chat_messages, contacts | (none found) | ? |
| POST /api/suggestion/choice | SuggestionChoiceHandler | (action record) | — | (none found) | ? |
| GET/PUT /api/about-me | AboutMeHandler | about_me | — | profileAPI, agentClient | keep |
| GET /api/context | ContextHandler | — | about_me, contacts, interactions | agentClient | keep |
| GET/POST /api/conversations | ConversationsHandler | conversations | chat_messages | profileAPI, backendConfig | keep |
| DELETE /api/conversations/{id} | ConversationsHandler | conversations, chat rows | — | profileAPI | keep (full delete) |
| GET/POST /api/contacts | ContactsHandler | contacts | — | profileAPI, backendConfig | keep |
| GET/PUT/DELETE /api/contacts/{id} | ContactDetailHandler | contacts | — | profileAPI | keep |
| GET /api/messages | MessagesHandler | — | chat_messages, conversations | profileAPI | keep |
| GET /api/reflections | ReflectionsHandler | — | (manager) | profileAPI | keep |
| POST /api/reflections/approval | ReflectionApprovalHandler | (approval) | reflections | (none found) | ? |
| POST /api/conflicts/resolve | ConflictResolveHandler | (action record) | (manager) | backendConfig | keep |
| GET /api/conflicts | ConflictsHandler | — | (manager) | (none found) | ? |
| GET /api/questions | GetPreviousQuestionsHandler | — | (question history) | (none found) | ? |
| GET /api/questions/effectiveness | QuestionEffectivenessHandler | (effectiveness) | — | (none found) | ? |
| POST /api/conversations/analyze | AnalyzeConversationHandler | — | chat_messages | (none found) | ? |
| GET /api/metrics | MetricsHandler | — | (metrics repos) | profileAPI | keep |
| DELETE /api/user/delete | DeleteProfileHandler | (full user delete) | users | backendConfig | keep |
| GET /api/status | handleStatus | — | — | agentClient, backendManager | keep |
| POST /api/analyze-mode-shift | handleAnalyzeModeShift | — | — | (none found) | ? |
| POST /api/generate-questions | handleGenerateQuestions | — | — | (none found) | ? |
| GET /api/constitution-principles | handleGetPrinciples | — | — | (none found) | ? |
| POST /api/frontend-errors | handleFrontendErrors | (log) | — | errorReporter | keep |

## 2. Extension calls with no server route

| Called from | Endpoint | Server route | Note |
|---|---|---|---|
| agentClient, backendConfig | /api/phase5/process | none | the old name of message-processor; calls will fail |
| settings/Settings.tsx (providerManager) | /api/providers, /api/providers/{claude,ollama,openai} | none | provider settings cannot work |
| api/modeAwareLLM.ts, providerManager | /api/chatModes, /api/providerManager, /api/tags | none | dead module, or broken |

## 3. Extension panels

| Panel | Imports | Data source | Decision |
|---|---|---|---|
| ChatInterface | backendManager | /api/message-processor via agentClient (status check only in backendManager) | keep |
| ConversationHistoryPanel | backendManager | /api/conversations, /api/messages | keep |
| AboutMeModal | profileAPI | /api/about-me | keep |
| ReflectionsPanel | backendManager | /api/reflections | keep |
| MetricsPanel | backendManager | /api/metrics (not called from this panel's imports) | ? |
| BehavioralInsightsPanel | backendManager | no endpoint of its own | ? |
| BackendStatus | backendManager | /api/status | keep |
| Settings | backendConfig, providerManager | provider routes do not exist | ? |
| Sidebar | backendConfig, clarificationAPI, profileAPI | — | keep |

## 4. What the message handler writes (the features that depend on it)

chat_messages, conversations, contacts, about_me, context_attributes, reflections, safety_incidents, system_context, conversation_execution_state.
If the rebuilt commit stage does not write one of these, the feature that reads it goes silent with no error. Rows marked "keep" above depend on this list.

## 5. Decisions needed from you

1. The rows marked **?**: keep, rebuild or drop each one.
2. The extension calls with no route: fix the caller, or drop the feature.
3. The panels marked **?**: do they show anything today? Please check them in the running app.

## 6. Limits of this inventory

- Writes and reads come from the handler's own SQL and repository calls. Where a handler calls a repository, I did not trace the repository's SQL, so rows marked "(manager)" or "(metrics repos)" are incomplete.
- "Used by" comes from imports and endpoint strings in the extension. A panel can get its data another way, which is why some panels are marked **?**.
- Nothing here has been run. The live check of each panel is still needed.
