package main

import (
	"regexp"
	"strings"
	"testing"

	"moly/database"
)

// Scenarios of memory mode (MOLY_MEMORY=1, docs/MEMORY_REDESIGN.md, step 5b and 5d): the memory update decides who the people are and
// what is asked. The model is scripted: the update call answers from the message it is shown.

func memoryModeServer(t *testing.T, update func(prompt string) string) (*APIServer, *database.Database, string, string) {
	t.Helper()
	return scenarioServer(t, scriptedLLM{
		gaps:         func() (string, error) { return "[]", nil },
		memoryUpdate: update,
	})
}

func lastMessage(prompt string) string {
	i := strings.LastIndex(prompt, "The new message from the user:\n")
	if i < 0 {
		return ""
	}
	return prompt[i:]
}

// M1: a person described by a role who has no name is added by the update, the name is asked, and the answer names the same person.
func TestMemoryModeAskedNameIsGivenToThePersonAsked(t *testing.T) {
	srv, db, userID, token := memoryModeServer(t, func(prompt string) string {
		switch msg := lastMessage(prompt); {
		case strings.Contains(msg, "talk to his father"):
			return `{"ops": [{"op":"add_person","label":"father","evidence":"his father"}], "next": {"action":"ask","kind":"name","about":"new1","topic":"p1 name","changes":"who to address"}}`
		case strings.Contains(msg, "Alex"):
			return `{"ops": [{"op":"name_person","person":"p1","name":"Alex","evidence":"Alex"}], "next": {"action":"answer"}}`
		}
		return `{"ops": [], "next": {"action":"answer"}}`
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I want to ask for advice about my job.")
	reply, _, status, _ := sendFull(t, srv, token, convID, "Should I talk to his father first?", false)
	if !strings.Contains(reply, "name for the father") {
		t.Fatalf("a new person without a name is asked for (status %d), got reply %q", status, reply)
	}
	reply, _, _, _ = sendFull(t, srv, token, convID, "Alex", false)
	if strings.Contains(reply, "name for the") {
		t.Fatalf("the name was given, it is not asked again, got %q", reply)
	}
	rows := contactsFor(t, db, userID)
	named := 0
	for _, r := range rows {
		if r.Name == "Alex" && r.NameStatus == "named" {
			named++
		}
		if r.Name == "father" {
			t.Fatalf("the person who was asked about has the name now, no 'father' row remains: %+v", rows)
		}
	}
	if named != 1 {
		t.Fatalf("one person called Alex, got %+v", rows)
	}
}

// M2: the update proposes a question on a topic the memory already holds as answered: the question discipline answers instead.
func TestMemoryModeAQuestionOnAnAnsweredTopicIsNotAsked(t *testing.T) {
	asked := 0
	srv, _, _, token := memoryModeServer(t, func(prompt string) string {
		if strings.Contains(lastMessage(prompt), "about the schedule") {
			asked++
			return `{"ops": [], "next": {"action":"ask","kind":"gap","topic":"schedule","missing":"the schedule","changes":"what to write"}}`
		}
		return `{"ops": [], "next": {"action":"answer"}}`
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I want to write a message to my team.")
	_, _, _, md := sendFull(t, srv, token, convID, "It is about the schedule.", false)
	if exit, _ := md["replyExit"].(string); exit != "gap_question" {
		t.Fatalf("the first time the question is asked, got exit %q", exit)
	}
	_, _, _, md = sendFull(t, srv, token, convID, "More about the schedule, as I said.", false)
	if exit, _ := md["replyExit"].(string); exit == "gap_question" {
		t.Fatalf("the same topic is not asked twice, got exit %q", exit)
	}
}

// M3: a message that states a person with a name and a role is stored once, with the role, and the contacts show it.
func TestMemoryModeProjectsThePeopleIntoTheContacts(t *testing.T) {
	srv, db, userID, token := memoryModeServer(t, func(prompt string) string {
		if strings.Contains(lastMessage(prompt), "my boss Kyle") {
			return `{"ops": [{"op":"add_person","name":"Kyle","evidence":"Kyle"},{"op":"set_role","person":"new1","role":"boss","of":"user","evidence":"my boss"}], "next": {"action":"answer"}}`
		}
		return `{"ops": [], "next": {"action":"answer"}}`
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I want help with a problem at work with my boss Kyle.")
	rows := contactsFor(t, db, userID)
	if len(rows) != 1 || rows[0].Name != "Kyle" || rows[0].NameStatus != "named" {
		t.Fatalf("one named person, Kyle, got %+v", rows)
	}
}

// M4: in memory mode the Skip button is offered once a question was answered (the memory's questions make the maturity), and the press
// marks the open questions skipped and gives the result.
func TestMemoryModeSkipIsOfferedAfterAnAnswerAndGivesTheResult(t *testing.T) {
	srv, _, _, token := memoryModeServer(t, func(prompt string) string {
		switch msg := lastMessage(prompt); {
		case strings.Contains(msg, "write to my team"):
			return `{"ops": [{"op":"set_goal","goal":"write to my team","evidence":"write to my team"}], "next": {"action":"ask","kind":"gap","topic":"schedule","missing":"the schedule","changes":"what to write"}}`
		case strings.Contains(msg, "about the schedule"):
			q := regexp.MustCompile(`"questions":\[\{"id":"(q\d+)"`).FindStringSubmatch(prompt) // ids run on across the tests that share a database
			if q == nil {
				return `{"ops": [], "next": {"action":"answer"}}`
			}
			return `{"ops": [{"op":"close_question","question":"` + q[1] + `","state":"answered","evidence":"about the schedule"}], "next": {"action":"ask","kind":"gap","topic":"audience","missing":"who reads it","changes":"the tone"}}`
		}
		return `{"ops": [], "next": {"action":"answer"}}`
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	_, _, _, md := sendFull(t, srv, token, convID, "I want to write to my team.", false)
	if exit, _ := md["replyExit"].(string); exit != "gap_question" {
		t.Fatalf("a first question is asked, got %q", exit)
	}
	if skip, _ := md["canSkip"].(bool); skip {
		t.Fatalf("nothing is answered yet: no skip button")
	}
	_, _, _, md = sendFull(t, srv, token, convID, "It is about the schedule.", false)
	if exit, _ := md["replyExit"].(string); exit != "gap_question" {
		t.Fatalf("a second question is asked, got %q", exit)
	}
	if skip, _ := md["canSkip"].(bool); !skip {
		t.Fatalf("one question was answered: the skip button is offered, metadata %v", md)
	}
	_, _, _, md = sendFull(t, srv, token, convID, "Go ahead with what you have.", true)
	if exit, _ := md["replyExit"].(string); exit != "help" {
		t.Fatalf("the press gives the result, got %q", exit)
	}
}

// M5 (replaces S19 and S23): a fact the update is not sure of is not saved: the reply is one confirming question.
func TestMemoryModeADoubtfulPersonIsConfirmedNotSaved(t *testing.T) {
	srv, db, userID, token := memoryModeServer(t, func(prompt string) string {
		if strings.Contains(lastMessage(prompt), "Maybe I should write to Anna") {
			return `{"ops": [], "next": {"action":"confirm","kind":"that you mean your colleague Anna"}}`
		}
		return `{"ops": [], "next": {"action":"answer"}}`
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	reply, _, _, md := sendFull(t, srv, token, convID, "Maybe I should write to Anna.", false)
	if exit, _ := md["replyExit"].(string); exit != "confirm_question" {
		t.Fatalf("a doubtful fact is confirmed, got exit %q, reply %q", exit, reply)
	}
	if rows := contactsFor(t, db, userID); len(rows) != 0 {
		t.Fatalf("nobody is saved on a doubt, got %+v", rows)
	}
}

// M6 (replaces S28 and S29): a skipped question may come back once; asked again and ignored it is not asked a third time.
func TestMemoryModeASkippedQuestionComesBackOnce(t *testing.T) {
	srv, _, _, token := memoryModeServer(t, func(prompt string) string {
		msg := lastMessage(prompt)
		q := regexp.MustCompile(`"id":"(q\d+)"[^{}]*"state":"open"`).FindStringSubmatch(prompt)
		switch {
		case strings.Contains(msg, "write to my team"):
			return `{"ops": [{"op":"set_goal","goal":"write to my team","evidence":"write to my team"}], "next": {"action":"ask","kind":"gap","topic":"schedule","missing":"the schedule","changes":"what to write"}}`
		case strings.Contains(msg, "about the schedule") && q != nil:
			return `{"ops": [{"op":"close_question","question":"` + q[1] + `","state":"answered","evidence":"about the schedule"}], "next": {"action":"ask","kind":"gap","topic":"audience","missing":"who reads it","changes":"the tone"}}`
		case strings.Contains(msg, "Anything else"):
			return `{"ops": [], "next": {"action":"ask","kind":"gap","topic":"audience","missing":"who reads it","changes":"the tone"}}`
		}
		return `{"ops": [], "next": {"action":"answer"}}`
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I want to write to my team.")
	sendMessage(t, srv, token, convID, "It is about the schedule.")
	if _, _, _, md := sendFull(t, srv, token, convID, "Go ahead with what you have.", true); md["replyExit"] != "help" {
		t.Fatalf("the press gives the result, got %v", md["replyExit"])
	}
	_, _, _, md := sendFull(t, srv, token, convID, "Anything else to ask?", false)
	if md["replyExit"] != "gap_question" {
		t.Fatalf("the skipped question comes back once, got %v", md["replyExit"])
	}
	_, _, _, md = sendFull(t, srv, token, convID, "Anything else to ask, again?", false)
	if md["replyExit"] == "gap_question" {
		t.Fatalf("it is asked and ignored: not a third time, got %v", md["replyExit"])
	}
}
