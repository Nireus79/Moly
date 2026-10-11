package database

import (
	"path/filepath"
	"testing"
	"time"

	"moly/models"
)

// Found live: relevant contacts were capitalised words of the last two messages, so on the answer "Yes" the person
// the conversation was about (Dana) was not found. They are now the user's contacts named anywhere in the conversation.
func TestRelevantContactsAreThoseNamedInTheConversation(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "rel.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db)
	for _, c := range []*models.Contact{
		{UserID: "u1", Name: "Dana", Relationship: "professional", Status: "active", NameStatus: "named"},
		{UserID: "u1", Name: "Bob", Relationship: "friend", Status: "active", NameStatus: "named"},
		{UserID: "u1", Name: "Dan", Relationship: "friend", Status: "active", NameStatus: "named"},
	} {
		if err := repo.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	b := &AnalysisContextBuilder{db: db}
	got := b.extractRelevantContacts("u1", []models.Message{
		{Role: "user", Content: "Her name is dana. She gives me extra tasks."},
		{Role: "assistant", Content: "Would you like to focus on a fairer distribution of tasks with your manager?"},
		{Role: "user", Content: "Yes"},
	})
	if len(got) != 1 || got[0].Name != "Dana" {
		names := []string{}
		for _, c := range got {
			names = append(names, c.Name)
		}
		t.Fatalf("only Dana is in the conversation (Dan is not Dana, Bob is not mentioned), got %v", names)
	}
}

func TestMentionsNameIsWholeWordAndIgnoresCase(t *testing.T) {
	cases := []struct {
		text, name string
		want       bool
	}{
		{"talk to Dana today", "Dana", true},
		{"talk to dana, please", "Dana", true},
		{"Danae is coming", "Dana", false},
		{"my manager said no", "manager", true},
		{"nothing here", "Dana", false},
		{"Dana", "Dana", true},
	}
	for _, c := range cases {
		if got := mentionsName(c.text, c.name); got != c.want {
			t.Errorf("mentionsName(%q, %q) = %v, want %v", c.text, c.name, got, c.want)
		}
	}
}
