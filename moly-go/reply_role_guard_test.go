package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The calls that write what the user reads, or a question Moly asks, ask for RoleReply (so that the reply model serves them); the
// small checks (judges, extraction, safety, summaries) do not. This lists the files that ask for it. A new call that writes text
// for the user belongs in the list; a new judge does not. (Plan "models by task", step 1.)
var askForReplyRole = regexp.MustCompile(`Role:\s+(tools\.)?RoleReply\b`)

func TestReplyRoleCallSites(t *testing.T) {
	want := []string{
		"agents/clarification_agent.go",
		"agents/conversation_agent.go",
		"agents/gap_novelty.go",
		"agents/memory_update.go",
		"agents/premise_check.go",
		"agents/refusal_question.go",
		"agents/single_question.go",
		"tools/response_generator.go",
	}
	var got []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if askForReplyRole.Match(data) {
			got = append(got, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the files that ask for the reply role changed:\n got  %v\n want %v", got, want)
	}
}
