package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"moly/database"
	"moly/tools"
)

type titleLLM struct {
	answer string
	err    error
	calls  int
	prompt string
}

func (l *titleLLM) Call(_ context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	l.calls++
	l.prompt = req.UserPrompt
	if l.err != nil {
		return nil, l.err
	}
	return &tools.LLMResponse{Content: l.answer}, nil
}

func titleFixture(t *testing.T, id, name string) (*APIServer, *titleLLM, func() string) {
	t.Helper()
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	conn := db.GetConnection()
	now := time.Now().Unix()
	conn.Exec(`INSERT OR IGNORE INTO users (id, created_at, last_active) VALUES ('title_user', ?, ?)`, now, now)
	conn.Exec(`DELETE FROM conversations WHERE id = ?`, id)
	if _, err := conn.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES (?, 'title_user', ?, ?, ?)`, id, name, now, now); err != nil {
		t.Fatal(err)
	}
	titledGoal.Delete(id)
	llm := &titleLLM{}
	srv := newBareServer(db)
	srv.llmClient = llm
	read := func() string {
		var n string
		conn.QueryRow(`SELECT name FROM conversations WHERE id = ?`, id).Scan(&n)
		return n
	}
	return srv, llm, read
}

func TestConversationIsNamedAfterItsGoal(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c1", defaultConversationName)
	llm.answer = "\"Heating repair request\"\n"
	srv.nameConversation("title_user", "title_c1", "I need to ask my landlord to fix the heating before winter")
	if name() != "Heating repair request" {
		t.Fatalf("name = %q", name())
	}
	if !strings.Contains(llm.prompt, "landlord") {
		t.Fatal("the model was not given the goal")
	}
}

func TestSameGoalDoesNotCallTheModelAgain(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c2", defaultConversationName)
	llm.answer = "Heating repair"
	srv.nameConversation("title_user", "title_c2", "fix the heating")
	srv.nameConversation("title_user", "title_c2", "fix the heating")
	if llm.calls != 1 || name() != "Heating repair" {
		t.Fatalf("calls=%d name=%q", llm.calls, name())
	}
}

func TestTitleFollowsAConfirmedGoalSwitch(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c3", defaultConversationName)
	llm.answer = "Heating repair"
	srv.nameConversation("title_user", "title_c3", "fix the heating")
	llm.answer = "Notice to leave"
	srv.nameConversation("title_user", "title_c3", "give my landlord notice that I am leaving")
	if name() != "Notice to leave" {
		t.Fatalf("name = %q", name())
	}
}

func TestUnusableModelAnswerFallsBackToTheStartOfTheGoal(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c4", defaultConversationName)
	llm.answer = strings.Repeat("this is a long explanation instead of a title ", 12)
	srv.nameConversation("title_user", "title_c4", "I want to tell my manager that the workload is not shared fairly")
	if name() != "I want to tell my manager" {
		t.Fatalf("name = %q", name())
	}
	srv2, llm2, name2 := titleFixture(t, "title_c5", defaultConversationName)
	llm2.err = errors.New("model down")
	srv2.nameConversation("title_user", "title_c5", "ask for a refund")
	if name2() != "Ask for a refund" {
		t.Fatalf("name = %q", name2())
	}
}

func TestAnEarlierRunsTitleIsKeptUntilTheGoalChanges(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c6", "Heating repair") // a title from before this run
	llm.answer = "Should not be used"
	srv.nameConversation("title_user", "title_c6", "fix the heating")
	if name() != "Heating repair" || llm.calls != 0 {
		t.Fatalf("name=%q calls=%d", name(), llm.calls)
	}
	llm.answer = "Notice to leave"
	srv.nameConversation("title_user", "title_c6", "give notice")
	if name() != "Notice to leave" {
		t.Fatalf("name = %q", name())
	}
}

func TestNoGoalNoTitle(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c7", defaultConversationName)
	srv.nameConversation("title_user", "title_c7", "  ")
	if name() != defaultConversationName || llm.calls != 0 {
		t.Fatalf("name=%q calls=%d", name(), llm.calls)
	}
}

func TestAnotherUsersConversationIsNeverRenamed(t *testing.T) {
	srv, llm, name := titleFixture(t, "title_c8", defaultConversationName)
	llm.answer = "Hijacked"
	srv.nameConversation("someone_else", "title_c8", "any goal")
	if name() != defaultConversationName {
		t.Fatalf("name = %q", name())
	}
}

func TestCleanTitle(t *testing.T) {
	for in, want := range map[string]string{
		"Title: Rent increase.":                "Rent increase",
		"  “Ζητώ επιστροφή χρημάτων”  \nextra": "Ζητώ επιστροφή χρημάτων",
		"":                                     "",
	} {
		if got := cleanTitle(in); got != want {
			t.Errorf("cleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
	if got := cleanTitle(strings.Repeat("word ", 30)); got != "" {
		t.Errorf("a long answer was accepted: %q", got)
	}
	if got := clip(strings.Repeat("abcdefgh ", 20)); len([]rune(got)) > titleMaxRunes {
		t.Errorf("too long: %d", len([]rune(got)))
	}
}
