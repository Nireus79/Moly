package agents

import (
	"path/filepath"
	"testing"
	"time"

	"moly/database"
	"moly/tools"
)

// An asked gap question that the user has since answered must not be asked again by Layer 4.
func TestAnsweredGapIsNotAskedAgain(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "l4answered.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	conn := db.GetConnection()
	conn.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)

	gapText := "What is the name of the person you want to write to?"
	questions := database.NewClarificationQuestionRepository(db)
	if err := questions.SaveQuestion(&database.ClarificationQuestion{
		ID: "q1", UserID: "u1", ConversationID: "c1", ClarificationType: "gap",
		QuestionText: gapText, Priority: 2, Status: "active", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	chr := database.NewClarificationHistoryRepository(db)
	if err := chr.SetQuestionStatus("u1", "q1", "answered", now+1); err != nil {
		t.Fatal(err)
	}

	l4 := NewLayer4GapDetector(nil)
	l4.SetClarificationHistory(chr)
	gaps := []tools.Gap{
		{Type: "gap_name", Description: gapText},
		{Type: "gap_other", Description: "Which of her messages do you want to answer first?"},
	}
	kept := l4.filterAnsweredGapsFromHistory(gaps, "u1", "c1")
	if len(kept) != 1 || kept[0].Description == gapText {
		t.Fatalf("answered gap should be dropped and the other kept, got %+v", kept)
	}
}
