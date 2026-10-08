package agents

import (
	"path/filepath"
	"testing"

	"moly/database"
)

func TestLayer10SessionSurvivesRestart(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "l10.db"))
	if err != nil {
		t.Fatal(err)
	}
	before := NewLayer10PersistentQuestioning(nil, db)
	if err := before.SaveSession("u1", "c1", &PersistenceSession{
		QuestionCount:       2,
		PreviousAnswers:     []string{"first answer", "second answer"},
		HasAcknowledgedHarm: true,
	}); err != nil {
		t.Fatal(err)
	}

	after := NewLayer10PersistentQuestioning(nil, db)
	got := after.LoadOrCreateSession("u1", "c1")
	if got.QuestionCount != 2 || len(got.PreviousAnswers) != 2 || !got.HasAcknowledgedHarm {
		t.Fatalf("layer 10 state not restored: %+v", got)
	}
}
