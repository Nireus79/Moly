package agents

import (
	"log"
	"time"

	"moly/models"
)

// PersonRepo is what the identity stage needs from storage. *database.ContactRepository satisfies it.
type PersonRepo interface {
	GetByName(userID, name string) (*models.Contact, error)
	GetSoleNamedByRelationship(userID, relationship string) (*models.Contact, error)
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
	// ResolveLabel says who the words taken as a new unnamed person really refer to: a person the user already has, a new
	// person, or no person at all (a pronoun with no one to point to, words about something else). Nil counts every label
	// as a new person.
	ResolveLabel func(label string, known []string) PersonRef
	// MarkAsked records that the person is being asked for a name. Nil writes it straight to the repo.
	MarkAsked func(c *models.Contact) error
}

// PersonResult is the person state after this message.
type PersonResult struct {
	NameAnswered     bool            // this message answered a name question
	PendingNameLabel string          // a person saved without a name; the reply must ask for it
	Contact          *models.Contact // the person this message is about, when there is one
	Created          bool            // the person was created by this message
	Doubt            string          // a person Moly is not sure of: not saved, to be confirmed with the user
}

// ResolvePerson is the only place a message writes a contact row (ORCHESTRATOR_DESIGN.md step 2).
// In order: apply a name answer; reuse the person by name, or by the only named person of that kind
// when the message gives no name; create an unnamed or named person otherwise; mark an unnamed person as asked.
func ResolvePerson(repo PersonRepo, in PersonInput) PersonResult {
	var out PersonResult
	if in.Greeting || in.Context == nil {
		return out
	}
	ec := in.Context.Contact

	givenName := ""
	if ec != nil && ec.NameKnown {
		givenName = ec.Name
	}
	answered, err := repo.ApplyNameAnswer(in.UserID, givenName)
	if err != nil {
		log.Printf("[Identity] Name answer could not be applied: %v", err)
	}
	out.NameAnswered = answered

	if ec == nil || ec.Name == "" {
		return out
	}

	existing, err := repo.GetByName(in.UserID, ec.Name)
	if err != nil {
		log.Printf("[Identity] Could not look up %q: %v", ec.Name, err)
		return out
	}
	if existing == nil && !ec.NameKnown {
		// no name and no record with this label: the message refers to the only named person of this kind, if there is one
		if sole, soleErr := repo.GetSoleNamedByRelationship(in.UserID, ec.Relationship); soleErr == nil && sole != nil {
			existing = sole
		}
	}

	if existing == nil && in.AnswerOnly && !ec.NameKnown {
		return out
	}
	if existing == nil && !ec.NameKnown && in.ResolveLabel != nil {
		label := ec.Label
		if label == "" {
			label = ec.Name
		}
		var known []string
		if people, listErr := repo.GetByUserID(in.UserID); listErr == nil {
			for _, c := range people {
				if c.Status == "active" && c.Name != "" {
					known = append(known, c.Name)
				}
			}
		}
		switch ref := in.ResolveLabel(label, known); ref.Kind {
		case refNone:
			log.Printf("[Identity] The words taken as a person do not refer to one: no person saved")
			return out
		case refKnown:
			if c, getErr := repo.GetByName(in.UserID, ref.Name); getErr == nil && c != nil {
				log.Printf("[Identity] The words refer to a person the user already has")
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
	}

	applyContactTraits(repo, in, &out)

	// An unnamed person is asked for a name once, and not in the message that answered a name question.
	if !out.NameAnswered && !ec.NameKnown {
		saved, getErr := repo.GetByName(in.UserID, ec.Name)
		if getErr == nil && saved != nil && saved.NameStatus == "unnamed" {
			mark := func(c *models.Contact) error { return repo.SetNameStatus(in.UserID, c.ID, "asked") }
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
		return "named"
	}
	return "unnamed"
}

// applyContactTraits adds the traits the message gave about people. A trait list is keyed by the name or the
// label the user used. It is matched exactly (never by substring), merged with what is stored, and never replaces it.
func applyContactTraits(repo PersonRepo, in PersonInput, out *PersonResult) {
	ec := in.Context.Contact
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
