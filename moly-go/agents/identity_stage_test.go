package agents

import (
	"context"
	"strings"
	"testing"

	"moly/models"
)

// fakePersonRepo is an in-memory PersonRepo with the same rules as the contact table that matter here.
type fakePersonRepo struct {
	rows   []*models.Contact
	nextID int64
	saves  int
}

func (f *fakePersonRepo) find(name string) *models.Contact {
	for _, c := range f.rows {
		if c.Name == name && c.Status == "active" {
			return c
		}
	}
	return nil
}
func (f *fakePersonRepo) GetByName(userID, name string) (*models.Contact, error) {
	return f.find(name), nil
}
func (f *fakePersonRepo) GetSoleNamedByRelationship(userID, rel string) (*models.Contact, error) {
	var found *models.Contact
	for _, c := range f.rows {
		if c.Relationship == rel && c.NameStatus == "named" {
			if found != nil {
				return nil, nil
			}
			found = c
		}
	}
	return found, nil
}
func (f *fakePersonRepo) GetByUserID(userID string) ([]*models.Contact, error) { return f.rows, nil }
func (f *fakePersonRepo) Save(c *models.Contact) error {
	f.saves++
	if c.ID == 0 {
		f.nextID++
		c.ID = f.nextID
		f.rows = append(f.rows, c)
	}
	return nil
}
func (f *fakePersonRepo) ApplyNameAnswer(userID, given string) (bool, error) {
	answered := false
	for _, c := range f.rows {
		if c.NameStatus != "asked" {
			continue
		}
		answered = true
		if given != "" && given != c.Name {
			c.Name = given
		}
		c.NameStatus = "named"
	}
	return answered, nil
}
func (f *fakePersonRepo) SetNameStatus(userID string, id int64, status string) error {
	for _, c := range f.rows {
		if c.ID == id {
			c.NameStatus = status
		}
	}
	return nil
}

func ctxWith(name, label, rel string, nameKnown bool, conf float64) *models.ExtractedContext {
	n := name
	if n == "" {
		n = label
	}
	return &models.ExtractedContext{Contact: &models.ExtractedContact{Name: n, Label: label, Relationship: rel, NameKnown: nameKnown, Confidence: conf}}
}

func TestResolvePersonCreatesUnnamedAndAsks(t *testing.T) {
	repo := &fakePersonRepo{}
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "girl", "other", false, 0.85)})
	if len(repo.rows) != 1 || repo.rows[0].Name != "girl" || repo.rows[0].NameStatus != "asked" {
		t.Fatalf("want one person 'girl' asked, got %+v", repo.rows)
	}
	if out.PendingNameLabel != "girl" || !out.Created || out.NameAnswered {
		t.Fatalf("unexpected result %+v", out)
	}
}

func TestResolvePersonRenamesOnNameAnswer(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "girl", "other", false, 0.85)})
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Christine", "girl", "other", true, 0.9)})
	if len(repo.rows) != 1 || repo.rows[0].Name != "Christine" || repo.rows[0].NameStatus != "named" {
		t.Fatalf("want the same row renamed and named, got %+v", repo.rows)
	}
	if !out.NameAnswered || out.PendingNameLabel != "" || out.Created {
		t.Fatalf("unexpected result %+v", out)
	}
}

func TestResolvePersonReusesByName(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Anna", "", "friend", true, 0.9)})
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Anna", "", "friend", true, 0.9)})
	if len(repo.rows) != 1 || out.Created {
		t.Fatalf("a known name must reuse the row, got %d rows, created=%v", len(repo.rows), out.Created)
	}
}

func TestResolvePersonReusesTheOnlyNamedOfThatKind(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Christine", "girl", "other", true, 0.9)})
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "she", "other", false, 0.8)})
	if len(repo.rows) != 1 || out.Created || out.Contact == nil || out.Contact.Name != "Christine" {
		t.Fatalf("a nameless reference must reuse the only named person, got %+v", repo.rows)
	}
	if out.PendingNameLabel != "" {
		t.Fatalf("the reused person is named; nothing to ask")
	}
}

func TestResolvePersonSeveralNamedIsNotGuessed(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Christine", "girl", "other", true, 0.9)})
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Mara", "girl", "other", true, 0.9)})
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "she", "other", false, 0.8)})
	if len(repo.rows) != 3 {
		t.Fatalf("two named people of one kind: a nameless reference becomes a new unnamed person, got %d rows", len(repo.rows))
	}
}

func TestResolvePersonGreetingWritesNothing(t *testing.T) {
	repo := &fakePersonRepo{}
	out := ResolvePerson(repo, PersonInput{UserID: "u", Greeting: true, Context: ctxWith("", "girl", "other", false, 0.9)})
	if len(repo.rows) != 0 || repo.saves != 0 || out.Contact != nil {
		t.Fatalf("a greeting must write no person, got %+v", repo.rows)
	}
}

func TestResolvePersonLowConfidenceWritesNothing(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "girl", "other", false, 0.4)})
	if len(repo.rows) != 0 {
		t.Fatalf("a low-confidence person must not be saved, got %+v", repo.rows)
	}
}

func TestResolvePersonNameAnswerWithoutGivenNameKeepsLabel(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "girl", "other", false, 0.85)})
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: &models.ExtractedContext{}})
	if !out.NameAnswered || repo.rows[0].Name != "girl" || repo.rows[0].NameStatus != "named" {
		t.Fatalf("an answer with no name keeps the label and stops asking, got %+v", repo.rows)
	}
}

func TestResolvePersonFillsBlanksButNeverOverwrites(t *testing.T) {
	repo := &fakePersonRepo{}
	first := ctxWith("Anna", "", "", true, 0.9)
	ResolvePerson(repo, PersonInput{UserID: "u", Context: first})
	second := ctxWith("Anna", "", "friend", true, 0.9)
	second.Contact.Traits = []string{"warm"}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: second})
	if repo.rows[0].Relationship != "friend" || len(repo.rows[0].Characteristics) != 1 {
		t.Fatalf("blanks must be filled, got %+v", repo.rows[0])
	}
	third := ctxWith("Anna", "", "colleague", true, 0.9)
	third.Contact.Traits = []string{"cold"}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: third})
	if repo.rows[0].Relationship != "friend" || repo.rows[0].Characteristics[0] != "warm" {
		t.Fatalf("stored values must not be overwritten, got %+v", repo.rows[0])
	}
}

func TestResolvePersonMergesTraitsByExactNameOrLabel(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Christine", "girl", "other", true, 0.9)})
	c := ctxWith("Christine", "girl", "other", true, 0.9)
	c.ContactCharacteristics = map[string][]string{"Christine": {"playful"}, "girl": {"playful", "direct"}, "Chris": {"unrelated"}}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: c})
	got := repo.rows[0].Characteristics
	if len(got) != 2 || got[0] != "playful" || got[1] != "direct" {
		t.Fatalf("traits must merge without duplicates and ignore partial names, got %v", got)
	}
	if len(repo.rows) != 1 {
		t.Fatalf("a trait list must not create people, got %d rows", len(repo.rows))
	}
}

func TestResolvePersonAnswerOnlyIntroducesNoUnnamedPerson(t *testing.T) {
	repo := &fakePersonRepo{}
	out := ResolvePerson(repo, PersonInput{UserID: "u", AnswerOnly: true, Context: ctxWith("", "that", "other", false, 0.85)})
	if len(repo.rows) != 0 || out.PendingNameLabel != "" || out.Created {
		t.Fatalf("an answer must not create an unnamed person: rows %+v result %+v", repo.rows, out)
	}
	// an answer that gives a name is still a person
	out = ResolvePerson(repo, PersonInput{UserID: "u", AnswerOnly: true, Context: ctxWith("Anna", "Anna", "other", true, 0.9)})
	if len(repo.rows) != 1 || repo.rows[0].Name != "Anna" {
		t.Fatalf("a named person in an answer is saved: %+v", repo.rows)
	}
}

func TestResolvePersonResolvesTheWordsTakenAsAPerson(t *testing.T) {
	// words that are not a person: nobody is saved
	repo := &fakePersonRepo{}
	none := func(string, []string) PersonRef { return PersonRef{Kind: refNone} }
	out := ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: none, Context: ctxWith("", "a bit too formal", "other", false, 0.9)})
	if len(repo.rows) != 0 || out.PendingNameLabel != "" {
		t.Fatalf("words about a text are not a person: %+v %+v", repo.rows, out)
	}
	// a pronoun that points to a known person is that person, not a new one
	repo = &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Anna", "Anna", "professional", true, 0.9)})
	var gotKnown []string
	toAnna := func(label string, known []string) PersonRef {
		gotKnown = known
		return PersonRef{Kind: refKnown, Name: "Anna"}
	}
	out = ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: toAnna, Context: ctxWith("", "she", "other", false, 0.9)})
	if len(repo.rows) != 1 || out.Created || out.PendingNameLabel != "" || out.Contact == nil || out.Contact.Name != "Anna" {
		t.Fatalf("'she' is Anna: rows %+v result %+v", repo.rows, out)
	}
	if len(gotKnown) != 1 || gotKnown[0] != "Anna" {
		t.Fatalf("the judge is told who the user already has, got %v", gotKnown)
	}
	// a named person is never judged
	called := false
	ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: func(string, []string) PersonRef { called = true; return PersonRef{Kind: refNone} }, Context: ctxWith("Bob", "Bob", "other", true, 0.9)})
	if called || len(repo.rows) != 2 {
		t.Fatalf("a named person is saved without the judgement: called=%v rows=%+v", called, repo.rows)
	}
}

// Found live: when the person-reference judgement failed, the pronoun "her" was saved as a second person. A failed
// judgement is doubt: nothing is saved and the reply asks.
func TestUnjudgedLabelIsDoubtNotANewPerson(t *testing.T) {
	repo := &fakePersonRepo{}
	unsure := func(string, []string) PersonRef { return PersonRef{Kind: refUnsure} }
	out := ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: unsure, Context: ctxWith("", "her", "other", false, 0.9)})
	if out.Doubt == "" || out.Contact != nil || len(repo.rows) != 0 {
		t.Fatalf("no person saved and a doubt raised, got %+v rows=%d", out, len(repo.rows))
	}
	if ref := JudgePersonReference(context.Background(), fixedLLM{content: "not json"}, "her", "msg", nil); ref.Kind != refUnsure {
		t.Fatalf("an unreadable judgement is unsure, got %q", ref.Kind)
	}
	if ref := JudgePersonReference(context.Background(), fixedLLM{content: "```json\n{\"refersTo\": \"none\"}\n```"}, "her", "msg", nil); ref.Kind != refNone {
		t.Fatalf("a fenced answer is read, got %q", ref.Kind)
	}
}

func TestLinkerDoesNotInventARole(t *testing.T) {
	c := &models.Contact{Name: "Dana", Relationship: "professional"}
	NewWhatWhoLinker().LinkWhatToWho(c, &models.ExtractedContext{Intention: "ask for fairer tasks"})
	if c.ContactRole != "" || len(c.Dependencies) != 0 {
		t.Fatalf("no invented role or dependencies, got role=%q deps=%v", c.ContactRole, c.Dependencies)
	}
}

func TestGapPromptKnowsAPersonsRole(t *testing.T) {
	block, _ := peopleStatusBlock(&models.AnalysisContext{RelevantContacts: []models.Contact{{Name: "Dana", NameStatus: "named", ContactRole: "manager"}}})
	if !strings.Contains(block, "manager") || !strings.Contains(block, "do NOT ask what Dana's role") {
		t.Fatalf("the role is given and must not be asked, got %q", block)
	}
}
