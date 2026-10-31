package models

// The structured memory of a conversation (docs/MEMORY_REDESIGN.md, section 3). Ids are written "p<number>" for a person and
// "q<number>" for a question, so that a model can point to them; the user is always "user".

// MemoryRole is what a person is: to the user (Of "user") or to another person (Of "p<number>").
type MemoryRole struct {
	Role   string `json:"role"`
	Of     string `json:"of"`
	Former bool   `json:"former,omitempty"`
}

// MemoryPerson is one person of the memory.
type MemoryPerson struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Label      string       `json:"label"`
	NameStatus string       `json:"nameStatus"`
	Roles      []MemoryRole `json:"roles"`
	Traits     []string     `json:"traits"`
}

// MemoryQuestion is a question Moly asked.
type MemoryQuestion struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	About string `json:"about"`
	Text  string `json:"text"`
	Topic string `json:"topic"` // the matter it asks about ("p2 involvement"): a topic is not asked twice
	State string `json:"state"` // open, answered, skipped, declined
}

// MemoryFact is a decided fact, with the words it came from.
type MemoryFact struct {
	Fact     string `json:"fact"`
	Person   string `json:"person"` // "user" or a person id
	Evidence string `json:"evidence"`
}

// Memory is everything Moly remembers of one conversation.
type Memory struct {
	People    []MemoryPerson   `json:"people"`
	Goal      string           `json:"goal"`
	Questions []MemoryQuestion `json:"questions"`
	Facts     []MemoryFact     `json:"facts"`
}

// MemoryOp is one change the update call proposes. Only the fields of its kind are used. Evidence is the exact words, taken from the
// message or the last messages, that justify the change.
type MemoryOp struct {
	Op       string `json:"op"`
	Person   string `json:"person,omitempty"`
	Into     string `json:"into,omitempty"` // merge_person: the person that remains
	Label    string `json:"label,omitempty"`
	Name     string `json:"name,omitempty"`
	Role     string `json:"role,omitempty"`
	Of       string `json:"of,omitempty"`
	Former   bool   `json:"former,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Trait    string `json:"trait,omitempty"`
	Goal     string `json:"goal,omitempty"`
	Question string `json:"question,omitempty"`
	State    string `json:"state,omitempty"`
	Fact     string `json:"fact,omitempty"`
	Evidence string `json:"evidence,omitempty"`
}

// The operations.
const (
	OpAddPerson         = "add_person"
	OpNamePerson        = "name_person"
	OpMergePerson       = "merge_person"
	OpSetRole           = "set_role"
	OpCorrectRole       = "correct_role"
	OpAddTrait          = "add_trait"
	OpSetGoal           = "set_goal"
	OpProposeGoalSwitch = "propose_goal_switch"
	OpCloseQuestion     = "close_question"
	OpAddFact           = "add_fact"
)

// The states of a question.
const (
	QuestionOpen     = "open"
	QuestionAnswered = "answered"
	QuestionSkipped  = "skipped"
	QuestionDeclined = "declined"
)

// MemoryUser is the id of the user in roles, traits and facts.
const MemoryUser = "user"

// MemoryNext is what Moly does after the memory update (docs/MEMORY_REDESIGN.md, sections 3a and 12).
type MemoryNext struct {
	Action string `json:"action"` // answer, ask, confirm or result_now
	Kind   string `json:"kind"`   // for ask: name, gap, clarity; for confirm: what to confirm
	About  string `json:"about"`  // the person id the question is about, when there is one
	// The question discipline: the matter the question asks about, what exactly is missing, and what the answer would change in the reply.
	// Code refuses a question with no stated effect and one on a topic already asked (agents.GuardNext).
	Topic   string `json:"topic"`
	Missing string `json:"missing"`
	Changes string `json:"changes"`
	// Voices and UserIs describe text the user pasted or retyped from memory that holds a talk between people (any format): how many different
	// people speak in it, and which of them is the user ("p?" of the memory, "unknown" when nothing says, "none" when the user is not in it).
	// Code, not the model, decides what follows: agents.GuardNext asks who is who when two or more speak and the user is unknown.
	Voices int    `json:"voices"`
	UserIs string `json:"user_is"`
	// Returning is set by the question discipline: the topic was skipped once, and the question comes back on purpose.
	Returning bool `json:"-"`
}
