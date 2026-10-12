package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"moly/privacy"
)

// The log holds no user content. A scripted conversation is run with the standard logger writing through the scrubber (as in main),
// and none of the words the user, the extraction and the replies carried may appear in what was logged. Before the scrubber the
// log quoted the goal, the names and the traits on dozens of lines (found while auditing profile deletion).
func TestLogsHoldNoUserContent(t *testing.T) {
	var buf bytes.Buffer
	privacy.Forget()
	s := privacy.NewScrubber(&buf)
	privacy.Default = s
	log.SetOutput(s)
	defer log.SetOutput(os.Stderr)
	srv, _, _, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	sendMessage(t, srv, token, convID, "SWITCH-WITH-NEW-PERSON Now I need to write to my brother about the party.")
	out := strings.ToLower(buf.String())
	if len(out) < 1000 {
		t.Fatalf("the test needs the log of the turns, got %d bytes", len(out))
	}
	for _, w := range []string{"fetlife", "christine", "write to my brother", "initiate a conversation"} {
		if n := strings.Count(out, w); n != 0 {
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, w) {
					t.Logf("leak: %.160s", line)
					break
				}
			}
			t.Errorf("%q appears %d time(s) in the log", w, n)
		}
	}
}
