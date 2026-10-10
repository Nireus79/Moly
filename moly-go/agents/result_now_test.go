package agents

import (
	"strings"
	"testing"

	"moly/models"
)

func TestResultNowRequestBlockCarriesTheRequestNotThePress(t *testing.T) {
	history := []models.Message{ // [current, oldest ... newest]
		{Role: "user", Content: "Go ahead with what you have."},
		{Role: "user", Content: "Write a warm thank-you note to Anna."},
		{Role: "assistant", Content: "How formal should it be?"},
		{Role: "user", Content: "Informal."},
		{Role: "assistant", Content: "What does she like?"},
	}
	got := resultNowRequestBlock(history)
	if !strings.Contains(got, "Write a warm thank-you note to Anna.") || !strings.Contains(got, "Moly: What does she like?") {
		t.Fatalf("the request and Moly's turns are shown: %q", got)
	}
	if strings.Contains(got, "Go ahead with what you have.") {
		t.Fatalf("the press itself is left out: %q", got)
	}
	if resultNowRequestBlock(history[:1]) != "" {
		t.Fatal("no earlier turns, no block")
	}
}
