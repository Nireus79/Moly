package agents

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"moly/models"
)

// fakePersonRepo is an in-memory PersonRepo of ONE conversation, with the same rules as the contact table that matter here:
// named people belong to the user, unnamed people to the conversation.
type fakePersonRepo struct {
	rows   []*models.Contact
	nextID int64
	saves  int
	// relations records every AddRelation as "person:relation:of:former" (of 0 = the user)
	relations []string
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
	if c, _ := f.GetNamedByName(userID, name); c != nil {
		return c, nil
	}
	return f.GetUnresolvedByLabel(userID, name)
}
func (f *fakePersonRepo) GetNamedByName(userID, name string) (*models.Contact, error) {
	for _, c := range f.rows {
		if c.Name == name && c.Status == "active" && c.NameStatus == "named" {
			return c, nil
		}
	}
	return nil, nil
}
func (f *fakePersonRepo) GetUnresolvedByLabel(userID, label string) (*models.Contact, error) {
	for _, c := range f.rows {
		if c.Name == label && c.Status == "active" && c.NameStatus != "named" {
			return c, nil
		}
	}
	return nil, nil
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
func (f *fakePersonRepo) ApplyNameAnswer(userID, given, forLabel string, declined bool) (bool, error) {
	answered := false
	for i := 0; i < len(f.rows); i++ {
		c := f.rows[i]
		if c.NameStatus != "asked" || (forLabel != "" && c.Name != forLabel) {
			continue
		}
		switch {
		case given != "" && given != c.Name:
			answered = true
			if known, _ := f.GetNamedByName(userID, given); known != nil {
				f.rows = append(f.rows[:i], f.rows[i+1:]...) // the person is the one the user already has: merged
				i--
				continue
			}
			c.Name, c.NameStatus = given, "named"
		case declined || given == c.Name:
			answered = true
			c.NameStatus = "nameless"
		}
	}
	return answered, nil
}
func (f *fakePersonRepo) AddRelation(userID string, id int64, relation string, ofID int64, former bool) error {
	f.relations = append(f.relations, fmt.Sprintf("%d:%s:%d:%t", id, relation, ofID, former))
	return nil
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

// The name first, the comparison after: a description ("she", "the girl", "my boss") is never matched with a saved person by
// its kind. Without an established person to point to, it is a new person of this conversation and the name is asked.
func TestResolvePersonDoesNotMatchADescriptionToASavedPerson(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Christine", "girl", "other", true, 0.9)})
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "girl", "other", false, 0.8)})
	if len(repo.rows) != 2 || !out.Created || out.PendingNameLabel != "girl" {
		t.Fatalf("a description must become a new unnamed person that is asked for, got %+v / %+v", repo.rows, out)
	}
}

// A person the conversation has already brought up can be pointed to by a role or a pronoun; the judge is told only those.
func TestResolvePersonPointsOnlyToEstablishedPeople(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Dana", "", "professional", true, 0.9)})
	var offered []string
	toDana := func(label string, known []string) PersonRef {
		offered = known
		if len(known) == 1 && known[0] == "Dana" {
			return PersonRef{Kind: refKnown, Name: "Dana"}
		}
		return PersonRef{Kind: refNew}
	}
	// the conversation has not brought Dana up: "my boss" is a new person, Dana is not even offered
	out := ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: toDana, Context: ctxWith("", "boss", "professional", false, 0.9)})
	if len(offered) != 0 || !out.Created || out.Contact == nil || out.Contact.Name != "boss" {
		t.Fatalf("an unestablished saved person must not be offered or matched: offered %v result %+v", offered, out)
	}
	// the conversation has brought Dana up: "she" is Dana
	repo2 := &fakePersonRepo{}
	ResolvePerson(repo2, PersonInput{UserID: "u", Context: ctxWith("Dana", "", "professional", true, 0.9)})
	out = ResolvePerson(repo2, PersonInput{UserID: "u", Established: []string{"Dana"}, ResolveLabel: toDana, Context: ctxWith("", "she", "other", false, 0.9)})
	if len(repo2.rows) != 1 || out.Created || out.Contact == nil || out.Contact.Name != "Dana" || out.PendingNameLabel != "" {
		t.Fatalf("'she' is the established Dana: %+v / %+v", repo2.rows, out)
	}
}

// A pronoun in the name field is not a name. It must not become a person, and must not answer a name question.
func TestAPronounInTheNameFieldIsNotAName(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "manager", "professional", false, 0.9)}) // asks for a name
	if len(repo.rows) != 1 || repo.rows[0].NameStatus != "asked" {
		t.Fatalf("setup: the manager is asked for, got %+v", repo.rows)
	}
	none := func(string, []string) PersonRef { return PersonRef{Kind: refNone} }
	declines := func([]string) NameOutcome { return NameOutcome{Kind: nameDeclined} }
	out := ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: none, JudgeNameAnswer: declines, Context: ctxWith("her", "her", "professional", true, 0.95)})
	for _, c := range repo.rows {
		if c.Name == "her" {
			t.Fatalf("'her' was saved as a person: %+v", repo.rows)
		}
	}
	if repo.rows[0].Name != "manager" || repo.rows[0].NameStatus != "nameless" || !out.NameAnswered {
		t.Fatalf("the question is closed without a name and the label stays, got %+v", repo.rows[0])
	}
}

// The name that arrives is compared with the people the user already has: the same name is the same person.
func TestTheNameGivenIsComparedWithSavedPeople(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Dana", "", "professional", true, 0.9)})
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "boss", "professional", false, 0.9)})
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Dana", "", "professional", true, 0.9)})
	if len(repo.rows) != 1 || repo.rows[0].Name != "Dana" || !out.NameAnswered || out.Created {
		t.Fatalf("the boss is Dana, merged into one person, got %+v / %+v", repo.rows, out)
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
	declines := func([]string) NameOutcome { return NameOutcome{Kind: nameDeclined} }
	out := ResolvePerson(repo, PersonInput{UserID: "u", JudgeNameAnswer: declines, Context: &models.ExtractedContext{}})
	if !out.NameAnswered || repo.rows[0].Name != "girl" || repo.rows[0].NameStatus != "nameless" {
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
	out = ResolvePerson(repo, PersonInput{UserID: "u", Established: []string{"Anna"}, ResolveLabel: toAnna, Context: ctxWith("", "she", "other", false, 0.9)})
	if len(repo.rows) != 1 || out.Created || out.PendingNameLabel != "" || out.Contact == nil || out.Contact.Name != "Anna" {
		t.Fatalf("'she' is Anna: rows %+v result %+v", repo.rows, out)
	}
	if len(gotKnown) != 1 || gotKnown[0] != "Anna" {
		t.Fatalf("the judge is told who the conversation has brought up, got %v", gotKnown)
	}
	// a name the user already has is that person and is not judged; a new name is judged (it may be a pronoun)
	called := 0
	count := func(string, []string) PersonRef { called++; return PersonRef{Kind: refNew} }
	ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: count, Context: ctxWith("Anna", "Anna", "other", true, 0.9)})
	if called != 0 || len(repo.rows) != 1 {
		t.Fatalf("a saved name needs no judgement: called=%d rows=%+v", called, repo.rows)
	}
	ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: count, Context: ctxWith("Bob", "Bob", "other", true, 0.9)})
	if called != 1 || len(repo.rows) != 2 {
		t.Fatalf("a new name is judged once and saved when it is a person: called=%d rows=%+v", called, repo.rows)
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
	if ref := JudgePersonReference(context.Background(), fixedLLM{content: "not json"}, "her", "msg", "", nil); ref.Kind != refUnsure {
		t.Fatalf("an unreadable judgement is unsure, got %q", ref.Kind)
	}
	if ref := JudgePersonReference(context.Background(), fixedLLM{content: "```json\n{\"refersTo\": \"none\"}\n```"}, "her", "msg", "", nil); ref.Kind != refNone {
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

func TestResolvePersonSavesEveryPersonOfOneMessage(t *testing.T) {
	repo := &fakePersonRepo{}
	ec := ctxWith("Dana", "manager", "professional", true, 0.9)
	ec.OtherContacts = []*models.ExtractedContact{{Name: "Mark", Label: "colleague", Relationship: "professional", NameKnown: true, Confidence: 0.9}}
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ec})
	if repo.find("Dana") == nil || repo.find("Mark") == nil {
		t.Fatalf("both people must be saved, got %+v", repo.rows)
	}
	if out.Contact == nil || out.Contact.Name != "Dana" {
		t.Fatalf("the main person stays the message's person: %+v", out.Contact)
	}
}

func TestResolvePersonAsksForTheNameOfAnotherPersonWhenTheMainOneIsNamed(t *testing.T) {
	repo := &fakePersonRepo{}
	ec := ctxWith("Dana", "manager", "professional", true, 0.9)
	ec.OtherContacts = []*models.ExtractedContact{{Name: "colleague", Label: "colleague", Relationship: "professional", Confidence: 0.9}}
	out := ResolvePerson(repo, PersonInput{UserID: "u", Context: ec})
	if out.PendingNameLabel != "colleague" {
		t.Fatalf("the unnamed second person is asked for, got %+v", out)
	}
}

func TestResolvePersonAnswerOnlyIntroducesNoSecondPerson(t *testing.T) {
	repo := &fakePersonRepo{}
	ec := ctxWith("Dana", "manager", "professional", true, 0.9)
	ec.OtherContacts = []*models.ExtractedContact{{Name: "Mark", NameKnown: true, Confidence: 0.9}}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ec, AnswerOnly: true})
	if repo.find("Mark") != nil {
		t.Fatal("an answer to a question introduces no one else")
	}
}

// Found in a live run: "My boss is unfair, my sister is getting married and my neighbour is spying on me" saved the boss and
// the sister; the model judging "my neighbour" was told the boss and the sister were people the user already has and
// matched it to the sister, so the neighbour was lost. People a message introduces are distinct from each other and are
// never offered as matches for each other.
func TestPeopleOfOneMessageAreNeverMatchedToEachOther(t *testing.T) {
	repo := &fakePersonRepo{}
	var offered [][]string
	matchTheFirst := func(label string, known []string) PersonRef {
		offered = append(offered, append([]string(nil), known...))
		if len(known) > 0 {
			return PersonRef{Kind: refKnown, Name: known[0]} // the mistake the model made
		}
		return PersonRef{Kind: refNew}
	}
	ec := ctxWith("", "boss", "professional", false, 0.9)
	ec.OtherContacts = []*models.ExtractedContact{
		{Name: "sister", Label: "sister", Relationship: "family", Confidence: 0.9},
		{Name: "neighbour", Label: "neighbour", Relationship: "other", Confidence: 0.8},
	}
	ResolvePerson(repo, PersonInput{UserID: "u", ResolveLabel: matchTheFirst, Context: ec})
	if len(repo.rows) != 3 {
		t.Fatalf("want boss, sister and neighbour saved, got %d people: %+v", len(repo.rows), repo.rows)
	}
	for i, known := range offered {
		if len(known) != 0 {
			t.Errorf("call %d was offered %v as people the user already has; the message itself introduced them", i+1, known)
		}
	}
	// the people the conversation has brought up before are what a later message may point to: the caller names them
	offered = nil
	ec2 := ctxWith("", "she", "romantic", false, 0.9)
	ResolvePerson(repo, PersonInput{UserID: "u", Established: []string{"boss", "sister", "neighbour"}, ResolveLabel: matchTheFirst, Context: ec2})
	if len(offered) != 1 || len(offered[0]) != 3 {
		t.Fatalf("earlier people are offered to a later message, got %v", offered)
	}
}

// The judge is shown "Dana (the user's boss)" and answers with the name; the stage gets the name alone.
func TestJudgeReturnsTheNameWithoutTheRole(t *testing.T) {
	ref := JudgePersonReference(context.Background(), fixedLLM{content: `{"refersTo": "known", "name": "Dana"}`}, "boss", "My boss is unfair.", "", []string{"Dana (the user's boss)"})
	if ref.Kind != refKnown || ref.Name != "Dana" {
		t.Fatalf("got %+v", ref)
	}
	ref = JudgePersonReference(context.Background(), fixedLLM{content: `{"refersTo": "known", "name": "Dana (the user's boss)"}`}, "boss", "My boss is unfair.", "", []string{"Dana (the user's boss)"})
	if ref.Kind != refKnown || ref.Name != "Dana" {
		t.Fatalf("an answer that repeats the role is read as the name, got %+v", ref)
	}
	ref = JudgePersonReference(context.Background(), fixedLLM{content: `{"refersTo": "known", "name": "Mara"}`}, "she", "She called.", "", []string{"Dana"})
	if ref.Kind != refUnsure {
		t.Fatalf("a person who is not in the list is never known: Moly asks, got %+v", ref)
	}
	// found live on mistral: "known" for "my boss" while nobody had been brought up made the user confirm a person never mentioned
	ref = JudgePersonReference(context.Background(), fixedLLM{content: `{"refersTo": "known", "name": "the boss"}`}, "boss", "My boss feels colder.", "", nil)
	if ref.Kind != refNew {
		t.Fatalf("known with nobody in the list is a new person, got %+v", ref)
	}
}

// Found live (2026-10-15): a message that does not answer the name question must leave it open; the name can come a turn later.
func TestAMessageThatIsNotAnAnswerLeavesThePersonWaiting(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "boss", "professional", false, 0.9)}) // asks for a name
	other := func([]string) NameOutcome { return NameOutcome{Kind: nameNotAnswered} }
	out := ResolvePerson(repo, PersonInput{UserID: "u", JudgeNameAnswer: other, Context: &models.ExtractedContext{}})
	if out.NameAnswered || repo.rows[0].NameStatus != "asked" {
		t.Fatalf("the question stays open, got %+v / %+v", out, repo.rows[0])
	}
	// with no judge at all the question also stays open (nothing closes it by accident)
	if out := ResolvePerson(repo, PersonInput{UserID: "u", Context: &models.ExtractedContext{}}); out.NameAnswered || repo.rows[0].NameStatus != "asked" {
		t.Fatalf("no judge: the question stays open, got %+v", repo.rows[0])
	}
	// and the name that arrives a turn later reaches the person, even when the extractor missed it and only the judge saw it
	gives := func(w []string) NameOutcome { return NameOutcome{Kind: nameGiven, Name: "Kyle"} }
	out = ResolvePerson(repo, PersonInput{UserID: "u", JudgeNameAnswer: gives, Context: &models.ExtractedContext{}})
	if !out.NameAnswered || repo.rows[0].Name != "Kyle" || repo.rows[0].NameStatus != "named" {
		t.Fatalf("the boss is Kyle, got %+v / %+v", out, repo.rows[0])
	}
}

// Two people wait: the name goes to the one the judge says it is for; an answer that does not say which names nobody.
func TestANameGoesToTheWaitingPersonItIsFor(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "boss", "professional", false, 0.9)})
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "father", "family", false, 0.9)})
	for _, c := range repo.rows {
		c.NameStatus = "asked"
	}
	forFather := func([]string) NameOutcome { return NameOutcome{Kind: nameGiven, Name: "Alex", For: "father"} }
	ResolvePerson(repo, PersonInput{UserID: "u", JudgeNameAnswer: forFather, Context: &models.ExtractedContext{}})
	var boss, alex *models.Contact
	for _, c := range repo.rows {
		if c.Name == "boss" {
			boss = c
		}
		if c.Name == "Alex" {
			alex = c
		}
	}
	if boss == nil || boss.NameStatus != "asked" || alex == nil || alex.NameStatus != "named" {
		t.Fatalf("Alex is the father; the boss still waits: %+v", repo.rows)
	}
}

// A label that is exactly the name of a person the conversation has already brought up is that person, even when the extractor took the
// name (a handle, a nickname) for a description (found live, 2026-10-15: a second nameless person and a name question for a named one).
func TestALabelEqualToAnEstablishedNameIsThatPerson(t *testing.T) {
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("PleasurePuppet74", "", "romantic", true, 0.9)})
	out := ResolvePerson(repo, PersonInput{
		UserID:      "u",
		Context:     ctxWith("PleasurePuppet74", "", "romantic", false, 0.9),
		Established: []string{"PleasurePuppet74 (romantic)"},
	})
	if len(repo.rows) != 1 || out.Created || out.PendingNameLabel != "" {
		t.Fatalf("want the one person reused and no name question, got %d rows, created=%v, pending=%q", len(repo.rows), out.Created, out.PendingNameLabel)
	}
	// without being established in this conversation the label is not matched to a saved person
	other := &fakePersonRepo{}
	ResolvePerson(other, PersonInput{UserID: "u", Context: ctxWith("Anna", "", "friend", true, 0.9)})
	ResolvePerson(other, PersonInput{UserID: "u", Context: ctxWith("Anna", "", "friend", false, 0.9)})
	if len(other.rows) != 2 {
		t.Fatalf("a saved person this conversation has not brought up is not matched by a label, got %d rows", len(other.rows))
	}
}

// A person Moly is unsure of is confirmed with the user only when the conversation has brought up nobody; with people already
// established the doubtful mention is dropped without a question (found live, 2026-10-15: four "do you mean ...?" in one conversation).
func TestADoubtfulMentionIsNotAskedAboutWhenPeopleAreEstablished(t *testing.T) {
	judge := func(label string, known []string) PersonRef { return PersonRef{Kind: refNew} }
	alone := ResolvePerson(&fakePersonRepo{}, PersonInput{UserID: "u", Context: ctxWith("", "someone", "other", false, 0.3), ResolveLabel: judge})
	if alone.Doubt == "" {
		t.Fatal("with nobody brought up, a doubtful person is confirmed")
	}
	repo := &fakePersonRepo{}
	ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("Anna", "", "friend", true, 0.9)})
	with := ResolvePerson(repo, PersonInput{UserID: "u", Context: ctxWith("", "someone", "other", false, 0.3), Established: []string{"Anna"}, ResolveLabel: judge})
	if with.Doubt != "" || len(repo.rows) != 1 {
		t.Fatalf("with a person established the doubtful mention is dropped silently, got doubt %q and %d rows", with.Doubt, len(repo.rows))
	}
}
