package privacy

import (
	"bytes"
	"strings"
	"testing"
)

func TestScrubberMasksRegisteredContentOnly(t *testing.T) {
	var buf bytes.Buffer
	s := NewScrubber(&buf)
	s.off = false
	s.Register("initiate a conversation with the girl", "Christine", "ab", "  ")
	s.Write([]byte(`[Layer1] Goal locked: "initiate a conversation with the girl" for Christine (ab) ok` + "\n"))
	got := buf.String()
	if strings.Contains(got, "Christine") || strings.Contains(got, "initiate") {
		t.Fatalf("registered content is masked: %q", got)
	}
	if !strings.Contains(got, "[Layer1] Goal locked") || !strings.Contains(got, "(ab)") {
		t.Fatalf("the rest of the line is kept, and a too-short string is not registered: %q", got)
	}
	s.Forget()
	buf.Reset()
	s.Write([]byte("Christine again\n"))
	if !strings.Contains(buf.String(), "Christine") {
		t.Fatalf("after Forget nothing is masked")
	}
}

func TestScrubberCanBeSwitchedOffForDebugging(t *testing.T) {
	var buf bytes.Buffer
	s := NewScrubber(&buf)
	s.off = true
	s.Register("secret goal")
	s.Write([]byte("the secret goal\n"))
	if !strings.Contains(buf.String(), "secret goal") {
		t.Fatal("MOLY_LOG_CONTENT=1 shows the content")
	}
}

func TestLongerPhrasesAreMaskedBeforeTheWordsInsideThem(t *testing.T) {
	var buf bytes.Buffer
	s := NewScrubber(&buf)
	s.off = false
	s.Register("Dana", "tell Dana I am late")
	s.Write([]byte("goal: tell Dana I am late\n"))
	if got := buf.String(); strings.Count(got, "[…]") != 1 {
		t.Fatalf("the phrase is masked as one piece: %q", got)
	}
}
