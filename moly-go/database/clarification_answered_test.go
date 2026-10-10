package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestOpenGapQuestionIsClosedWhenAnswered(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "clar.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	conn := db.GetConnection()
	conn.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)

	questions := NewClarificationQuestionRepository(db)
	gapText := "What do you want her to feel after reading your first message?"
	if err := questions.SaveQuestion(&ClarificationQuestion{
		ID: "q1", UserID: "u1", ConversationID: "c1", ClarificationType: "gap",
		QuestionText: gapText, Priority: 2, Status: "active", CreatedAt: now,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	chr := NewClarificationHistoryRepository(db)
	if answered, _ := chr.HasQuestionTextBeenAnswered("u1", "c1", gapText); answered {
		t.Fatal("question must not be answered before the user replies")
	}

	open, err := chr.OpenQuestions("u1", "c1")
	if err != nil || len(open) != 1 || open[0].Status != "active" {
		t.Fatalf("the asked question is open: %+v err=%v", open, err)
	}
	if err := chr.SetQuestionStatus("u1", open[0].ID, "answered", now+1); err != nil {
		t.Fatal(err)
	}
	if answered, _ := chr.HasQuestionTextBeenAnswered("u1", "c1", gapText); !answered {
		t.Fatal("question should be answered after the user's reply")
	}
	texts, err := chr.AnsweredQuestionTexts("u1", "c1", 20)
	if err != nil || len(texts) != 1 || texts[0] != gapText {
		t.Fatalf("answered list: %v err=%v", texts, err)
	}
}

func TestOpenQuestionsAreScopedToUserAndConversation(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "clar2.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	conn := db.GetConnection()
	conn.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u2', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c2', 'u2', 'n', ?, ?)`, now, now)

	questions := NewClarificationQuestionRepository(db)
	questions.SaveQuestion(&ClarificationQuestion{ID: "q1", UserID: "u1", ConversationID: "c1", ClarificationType: "gap", QuestionText: "A", Priority: 2, Status: "active", CreatedAt: now})
	questions.SaveQuestion(&ClarificationQuestion{ID: "q2", UserID: "u2", ConversationID: "c2", ClarificationType: "gap", QuestionText: "B", Priority: 2, Status: "active", CreatedAt: now})

	chr := NewClarificationHistoryRepository(db)
	open, _ := chr.OpenQuestions("u1", "c1")
	if len(open) != 1 || open[0].Text != "A" {
		t.Fatalf("only the u1/c1 question is listed, got %+v", open)
	}
	// closing a question for the wrong user changes nothing
	chr.SetQuestionStatus("u1", "q2", "answered", now+1)
	if other, _ := chr.HasQuestionTextBeenAnswered("u2", "c2", "B"); other {
		t.Fatal("another user's open question was marked answered")
	}
}

// A question the user skipped is pending: not answered, not active, and only the ids given are touched (the earlier answer in the same second stays).
func TestSkippedQuestionsArePending(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "skip.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	conn := db.GetConnection()
	conn.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	questions := NewClarificationQuestionRepository(db)
	questions.SaveQuestion(&ClarificationQuestion{ID: "old", UserID: "u1", ConversationID: "c1", ClarificationType: "gap", QuestionText: "Old?", Priority: 2, Status: "answered", CreatedAt: now})
	questions.SaveQuestion(&ClarificationQuestion{ID: "q1", UserID: "u1", ConversationID: "c1", ClarificationType: "gap", QuestionText: "Low?", Priority: 2, Status: "active", CreatedAt: now})
	questions.SaveQuestion(&ClarificationQuestion{ID: "q2", UserID: "u1", ConversationID: "c1", ClarificationType: "goal", QuestionText: "High?", Priority: 1, Status: "active", CreatedAt: now})

	chr := NewClarificationHistoryRepository(db)
	open, _ := chr.OpenQuestions("u1", "c1")
	if len(open) != 2 {
		t.Fatalf("two open questions, got %+v", open)
	}
	for _, q := range open {
		// skipped in the same second as the old answer on purpose
		if err := chr.SetQuestionStatus("u1", q.ID, "skipped", 0); err != nil {
			t.Fatal(err)
		}
	}
	var answered, skipped int
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE status='answered'`).Scan(&answered)
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE status='skipped'`).Scan(&skipped)
	if answered != 1 || skipped != 2 {
		t.Fatalf("the earlier answer stays answered: answered=%d skipped=%d", answered, skipped)
	}
	if texts, _ := chr.AnsweredQuestionTexts("u1", "c1", 20); len(texts) != 1 {
		t.Fatalf("only the answered one is listed as answered, got %v", texts)
	}
	pending, _ := chr.PendingSkipped("u1", "c1")
	if len(pending) != 2 {
		t.Fatalf("both skipped questions are pending, got %+v", pending)
	}
	if again, _ := chr.ReactivateSkippedByText("u1", "c1", "High?"); !again {
		t.Fatal("asking a pending question again makes it active")
	}
	if err := chr.SetQuestionStatus("u1", "q1", "answered", now); err != nil {
		t.Fatal(err)
	}
	if n, _ := chr.CancelPending("u1", "c1"); n != 0 {
		t.Fatalf("nothing is pending any more, cancelled %d", n)
	}
}

// A reactivated question must stay readable: the store keeps 0, never NULL, for the answer time of an open question.
func TestReactivatedQuestionIsReadable(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "react.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	conn := db.GetConnection()
	conn.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	questions := NewClarificationQuestionRepository(db)
	questions.SaveQuestion(&ClarificationQuestion{ID: "q1", UserID: "u1", ConversationID: "c1", ClarificationType: "gap", QuestionText: "Q?", Priority: 2, Status: "skipped", CreatedAt: now, AnsweredAt: now})
	chr := NewClarificationHistoryRepository(db)
	if again, err := chr.ReactivateSkippedByText("u1", "c1", "Q?"); !again || err != nil {
		t.Fatalf("reactivate: %v %v", again, err)
	}
	got, err := questions.GetConversationQuestions("c1")
	if err != nil || len(got) != 1 || got[0].Status != "active" {
		t.Fatalf("the reactivated question is readable and active: %+v %v", got, err)
	}
}
