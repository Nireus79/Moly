package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCLIEnvDropsTheAPICredentials(t *testing.T) {
	env := claudeCLIEnv([]string{"PATH=/bin", "ANTHROPIC_API_KEY=secret", "ANTHROPIC_AUTH_TOKEN=x", "HOME=/h", "ANTHROPIC_MODEL=m"})
	joined := strings.Join(env, " ")
	if strings.Contains(joined, "secret") || strings.Contains(joined, "ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("the API credentials must not reach the child (it would bill an API account), got %v", env)
	}
	if !strings.Contains(joined, "PATH=/bin") || !strings.Contains(joined, "HOME=/h") || !strings.Contains(joined, "ANTHROPIC_MODEL=m") {
		t.Fatalf("the rest of the environment stays, got %v", env)
	}
}

// A stand-in `claude` command shows how it is called: the prompt on stdin, no tools, the chosen model, and no API key.
func TestClaudeCLICallsTheCommandWithoutToolsAndWithoutTheKey(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\ncat > \"" + dir + "/stdin.txt\"\necho \"ARGS: $*\" > \"" + dir + "/args.txt\"\necho \"KEY=[$ANTHROPIC_API_KEY]\" > \"" + dir + "/env.txt\"\necho 'the answer'\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "must-not-reach-the-child")
	c := &LLMClient{Provider: "claude-cli", Model: "haiku"}
	resp, err := c.callClaudeCLI(context.Background(), &LLMRequest{SystemPrompt: "SYSTEM TEXT", UserPrompt: "USER TEXT"})
	if err != nil || resp.Content != "the answer" {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
	read := func(name string) string { b, _ := os.ReadFile(filepath.Join(dir, name)); return string(b) }
	if !strings.Contains(read("stdin.txt"), "USER TEXT") {
		t.Fatal("the user prompt goes on stdin")
	}
	args := read("args.txt")
	for _, want := range []string{"-p", "--system-prompt SYSTEM TEXT", "--tools", "--no-session-persistence", "--model haiku"} {
		if !strings.Contains(args, want) {
			t.Fatalf("the command lacks %q: %s", want, args)
		}
	}
	if strings.Contains(read("env.txt"), "must-not-reach-the-child") {
		t.Fatal("the API key reached the child")
	}
	if _, err := c.callClaudeCLI(context.Background(), &LLMRequest{UserPrompt: "  "}); err == nil {
		t.Fatal("an empty prompt is an error")
	}
}
