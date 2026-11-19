package tools

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// The choice of models is written to the log with how each was found, so that a change in a provider's naming shows at once.
// The line holds model names only: never the key.
func TestChosenModelsAreLoggedWithHowTheyWereFoundAndNeverTheKey(t *testing.T) {
	restore := SetModelLister(func(context.Context, LLMSettings) (*ModelList, error) {
		return list("gpt-7-mini", "gpt-7"), nil
	})
	defer restore()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(log.Writer())
	const key = "sk-test-secret-key-123"
	roles, err := chooseRoleModels("openai", key, false)
	if err != nil || roles.Reply != "gpt-7-mini" {
		t.Fatalf("got %+v %v", roles, err)
	}
	out := buf.String()
	for _, want := range []string{"openai models", "fast=gpt-7-mini (family)", "reply=gpt-7-mini (family)", "tested=false"} {
		if !strings.Contains(out, want) {
			t.Errorf("the log lacks %q: %s", want, out)
		}
	}
	if strings.Contains(out, key) {
		t.Fatalf("the key is in the log: %s", out)
	}
}

// When nothing in the list fits the pattern the log says the provider's suggestion was used.
func TestALogLineSaysWhenTheProvidersSuggestionWasUsed(t *testing.T) {
	restore := SetModelLister(func(context.Context, LLMSettings) (*ModelList, error) {
		return list("o9", "gpt-7-pro"), nil
	})
	defer restore()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(log.Writer())
	if _, err := chooseRoleModels("openai", "k", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "(suggestion)") {
		t.Fatalf("got %s", buf.String())
	}
}
