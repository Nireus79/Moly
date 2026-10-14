package agents

import (
	"log"
	"time"

	"moly/models"
)

// PersonRepo is what the identity stage needs from storage. *database.ContactRepository satisfies it. It is the repository of
// one conversation: it sees the people the user has named and the unnamed people of this conversation, and no others.
type PersonRepo interface {
	GetByName(userID, name string) (*models.Contact, error)
	GetNamedByName(userID, name string) (*models.Contact, error)
	GetUnresolvedByLabel(userID, label string) (*models.Contact, error)
	GetByUserID(userID string) ([]*models.Contact, error)
	Save(contact *models.Contact) error
	ApplyNameAnswer(userID, givenName string) (bool, error)
	SetNameStatus(userID string, contactID int64, status string) error
}

// PersonInput is everything the stage reads. Nothing else decides who a message is about.
type PersonInput struct {
	UserID   string
	Context  *models.ExtractedContext
	Greeting bool // a greeting names no one and answers no question
	// AnswerOnly: the message only answers a question Moly asked. It may give a name or point to a known person,
	// but it never introduces a new unnamed person (a word in an answer is not a new person).
	AnswerOnly bool
	// NoNameQuestion: the message states no goal and none is locked (a chat, a feeling, a disclosure). A person is saved but not asked for a name
	// yet: the name is needed to write to someone, not to talk about them (found in a dynamic run: "can you give me a name for the grandmother").
	NoNameQuestion bool
	// Established are the people this conversation has already brought up before this message: the saved people it mentions by
	// name, and its own people who have no name yet (by label). A pronoun or a role ("she", "my boss") can only point to one of
	// them. A saved person whom this conversation has not brought up is never a candidate: a person is matched with the saved
	// ones by the name the user gives, not by the words "boss" or "her".
	Established []string
	// ResolveLabel says who the words taken as a person really refer to, among the established people: one of them, a new
	// person, or no person at all (a pronoun with no one to point to, words about something else). It judges a label and also a
	// name the model took for a real one ("her" is not a name). Nil counts every label and name as a new person.
	ResolveLabel func(label string, known []string) PersonRef
	// MarkAsked records that the person is being asked for a name. Nil writes it straight to the repo.
	MarkAsked func(c *models.Contact) error
	// TraitsChanged says whether new traits describe a change of a person against the traits stored BEFORE this message
	// (JudgeTraitChange). Nil counts nothing as a change: traits are only added.
	TraitsChanged func(before, incoming []string) bool
}

// TraitConflict is a change of a person the model found: the traits stored before this message and the ones it gave. The new
// traits are NOT added to the person; the user is asked, and the answer decides (use_extracted adds them, keep_saved drops them).
type TraitConflict struct {
	Name     string
	Before   []string
	Incoming []string
}

// PersonResult is the person state after this message.
type PersonResult struct {
	NameAnswered     bool            // this message answered a name question
	PendingNameLabel string          // a person saved without a name; the reply must ask for it
	Contact          *models.Contact // the person this message is about, when there is one
	Created          bool            // the person was created by this message
	Doubt            string          // a person Moly is not sure of: not saved, to be confirmed with the user
	TraitConflicts   []TraitConflict // traits that describe a change of a person; held back until the user answers
}

// ResolvePerson is the only place a message writes a contact row (ORCHESTRATOR_DESIGN.md step 2).
//
// The order is: the name first, the comparison after. A person the user names ("Dana") is compared with the people the user
// has. A person the user only describes ("my boss") is NOT matched with a saved person by that description: it is kept as a
// person of this conversation, known by the label, and the name is asked. When the name comes, it is compared with the saved
// people (NameContact): the same name is the same person, merged. Only a pronoun or a role can point to a person, and only to
// one this conversation has already brought up (PersonInput.Established).
//
// A message may mention several people (Contact and OtherContacts). Each is resolved the same way. The main person decides
// NameAnswered and Contact; Moly asks one thing per reply, so a doubt or a name question from another person is reported only
// when the main person has none.
func ResolvePerson(repo PersonRepo, in PersonInput) PersonResult {
	var out PersonResult
	if in.Greeting || in.Context == nil {
		return out
	}
	r := &resolver{repo: repo, in: in, verdicts: map[string]PersonRef{}}
	ec := in.Context.Contact

	// A name the user gave answers a name question only when it is a name: the model may put a pronoun in the name field.
	givenName := ""
	if ec != nil && ec.NameKnown {
		if saved, getErr := repo.GetNamedByName(in.UserID, ec.Name); getErr == nil && saved != nil {
			givenName = ec.Name // the name of a person the user already has
		} else {
			switch r.judge(ec.Name).Kind {
			case refNew:
				givenName = ec.Name
			case refUnsure:
				// not judged: the question stays open, and the reply asks who is meant
				out.Doubt = ec.Name
				return out
			}
		}
	}
	answered, err := repo.ApplyNameAnswer(in.UserID, givenName)
	if err != nil {
		log.Printf("[Identity] Name answer could not be applied: %v", err)
	}
	out.NameAnswered = answered

	sameMessage := map[string]bool{} // people this message itself introduced: distinct from each other, never a match for each other
	out = r.resolveOne(ec, out, false, sameMessage)
	if in.AnswerOnly { // an answer to a question points to a person Moly asked about; it introduces no one else
		return out
	}
	for _, other := range in.Context.OtherContacts {
		extra := r.resolveOne(other, PersonResult{NameAnswered: answered}, true, sameMessage)
		if out.Doubt == "" && out.PendingNameLabel == "" {
			out.Doubt = extra.Doubt
			out.PendingNameLabel = extra.PendingNameLabel
		}
		if out.Contact == nil {
			out.Contact = extra.Contact
		}
		out.Created = out.Created || extra.Created
		out.TraitConflicts = append(out.TraitConflicts, extra.TraitConflicts...)
	}
	return out
}

// resolver holds one message's resolution. The judge is asked once per set of words.
type resolver struct {
	repo     PersonRepo
	in       PersonInput
	verdicts map[string]PersonRef
}

// judge says what the words refer to among the established people. It is asked once per set of words in a message.
func (r *resolver) judge(words string) PersonRef {
	if v, ok := r.verdicts[words]; ok {
		return v
	}
	v := PersonRef{Kind: refNew}
	if r.in.ResolveLabel != nil {
		v = r.in.ResolveLabel(words, r.in.Established)
	}
	r.verdicts[words] = v
	return v
}

// resolveOne resolves one extracted person and returns the result with the state before it (out) updated.
//
// secondary marks a person mentioned next to the main one.
func (r *resolver) resolveOne(ec *models.ExtractedContact, out PersonResult, secondary bool, sameMessage map[string]bool) PersonResult {
	repo, in := r.repo, r.in
	if ec == nil || ec.Name == "" {
		return out
	}

	var existing *models.Contact
	var err error
	if ec.NameKnown {
		existing, err = repo.GetNamedByName(in.UserID, ec.Name) // a name is compared with every person the user has
	} else {
		existing, err = repo.GetUnresolvedByLabel(in.UserID, ec.Name) // a label only with this conversation's own people
	}
	if err != nil {
		log.Printf("[Identity] Could not look up %q: %v", ec.Name, err)
		return out
	}

	if existing == nil && in.AnswerOnly && !ec.NameKnown {
		return out
	}
	if existing == nil {
		words := ec.Name
		if !ec.NameKnown && ec.Label != "" {
			words = ec.Label
		}
		switch ref := r.judge(words); ref.Kind {
		case refNone:
			log.Printf("[Identity] The words taken as a person do not refer to one: no person saved")
			return out
		case refUnsure:
			out.Doubt = words
			log.Printf("[Identity] Could not judge who the words refer to: no person saved; the reply asks")
			return out
		case refKnown:
			if c, getErr := repo.GetByName(in.UserID, ref.Name); getErr == nil && c != nil {
				log.Printf("[Identity] The words refer to a person this conversation already has")
				existing = c
			}
		}
	}

	if existing == nil && FactInDoubt(ec.Confidence) {
		// A person Moly is not sure of is not saved. The caller asks the user to confirm who is meant.
		out.Doubt = ec.Name
		return out
	}

	linker := NewWhatWhoLinker()
	if existing != nil {
		linker.LinkWhatToWho(existing, in.Context)
		// A stored value is never overwritten here; only blanks are filled (changes go through the conflict check).
		if existing.Relationship == "" {
			existing.Relationship = ec.Relationship
		}
		if existing.ContactRole == "" {
			existing.ContactRole = ec.Role
		}
		if len(existing.Characteristics) == 0 {
			existing.Characteristics = ec.Traits
		}
		if err := repo.Save(existing); err != nil {
			log.Printf("[Identity] Could not update %q: %v", existing.Name, err)
		}
		out.Contact = existing
	} else {
		now := time.Now().Unix()
		created := &models.Contact{
			UserID:          in.UserID,
			Name:            ec.Name,
			NameStatus:      nameStatusOf(ec),
			Relationship:    ec.Relationship,
			ContactRole:     ec.Role,
			Characteristics: ec.Traits,
			Confidence:      ec.Confidence,
			CreatedVia:      "conversation",
			Status:          "active",
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		linker.LinkWhatToWho(created, in.Context)
		if err := repo.Save(created); err != nil {
			log.Printf("[Identity] Could not save %q: %v", ec.Name, err)
			return out
		}
		out.Contact = created
		out.Created = true
		sameMessage[created.Name] = true
	}

	applyContactTraits(repo, in, ec, &out)

	// An unnamed person is asked for a name once, and not in the message that answered a name question.
	if !out.NameAnswered && !ec.NameKnown && !in.NoNameQuestion {
		saved, getErr := repo.GetUnresolvedByLabel(in.UserID, ec.Name)
		if getErr == nil && saved != nil && saved.NameStatus == models.NameStatusUnnamed {
			mark := func(c *models.Contact) error { return repo.SetNameStatus(in.UserID, c.ID, models.NameStatusAsked) }
			if in.MarkAsked != nil {
				mark = in.MarkAsked
			}
			if setErr := mark(saved); setErr == nil {
				out.PendingNameLabel = saved.Name
				out.Contact = saved
			}
		}
	}
	return out
}

func nameStatusOf(c *models.ExtractedContact) string {
	if c != nil && c.NameKnown {
		return models.NameStatusNamed
	}
	return models.NameStatusUnnamed
}

// applyContactTraits adds the traits the message gave about people. A trait list is keyed by the name or the
// label the user used. It is matched exactly (never by substring), merged with what is stored, and never replaces it.
func applyContactTraits(repo PersonRepo, in PersonInput, ec *models.ExtractedContact, out *PersonResult) {
	for key, traits := range in.Context.ContactCharacteristics {
		if len(traits) == 0 {
			continue
		}
		target, err := repo.GetByName(in.UserID, key)
		if err != nil {
			continue
		}
		if target == nil && out.Contact != nil && ec != nil && (key == ec.Name || key == ec.Label) {
			target = out.Contact
		}
		if target == nil {
			continue
		}
		merged, added := mergeTraits(target.Characteristics, traits)
		if !added {
			continue
		}
		// Compared with what was stored before this message, never with what this message just wrote. A person who is described
		// as changed is not rewritten: the user is asked first.
		if len(target.Characteristics) > 0 && in.TraitsChanged != nil && in.TraitsChanged(target.Characteristics, traits) {
			out.TraitConflicts = append(out.TraitConflicts, TraitConflict{Name: target.Name, Before: append([]string(nil), target.Characteristics...), Incoming: traits})
			continue
		}
		target.Characteristics = merged
		if err := repo.Save(target); err != nil {
			log.Printf("[Identity] Could not save traits for %q: %v", target.Name, err)
		}
	}
}

func mergeTraits(stored, incoming []string) ([]string, bool) {
	have := make(map[string]bool, len(stored))
	for _, t := range stored {
		have[t] = true
	}
	merged := append([]string(nil), stored...)
	added := false
	for _, t := range incoming {
		if t != "" && !have[t] {
			have[t] = true
			merged = append(merged, t)
			added = true
		}
	}
	return merged, added
}
