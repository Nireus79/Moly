package agents

// CrisisReply is the whole reply to a message that shows explicit self-harm intent. It is a short refusal that names
// who to ask. It carries no help lines (they read as calling the user unwell), asks nothing, and is fixed text so it
// does not depend on the model that just judged the message. An unclear message never reaches it: the clarification
// and Socratic questions come first, and only an explicit statement is judged a crisis.
const CrisisReply = "I can't help with that. Please consider asking advice from a specialist."
