package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// The "claude-cli" provider is for TESTING Moly's logic with a strong model, fast, on the developer's own Claude login: every
// call runs `claude -p` (Claude Code in print mode) with no tools. It needs no API key; the child process is started without
// ANTHROPIC_API_KEY (and the other API credentials) so that it uses the claude.ai login and not an API account. The prompts leave
// the machine: use it with throwaway test users only. The temperature and the token limit of a request cannot be set through the
// command line, so they are ignored. LLM_PROVIDER=claude-cli, AGENT_MODEL=haiku|sonnet|opus (default haiku).

// claudeCLIEnv is the environment of the child process: the parent's, without the API credentials.
func claudeCLIEnv(parent []string) []string {
	env := make([]string, 0, len(parent))
	for _, kv := range parent {
		name := strings.SplitN(kv, "=", 2)[0]
		switch name {
		case "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN":
			continue
		}
		env = append(env, kv)
	}
	return env
}

func (c *LLMClient) callClaudeCLI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if strings.TrimSpace(req.UserPrompt) == "" {
		return nil, errors.New("user prompt cannot be empty")
	}
	system := req.SystemPrompt
	if strings.TrimSpace(system) == "" {
		system = "You are a helpful assistant."
	}
	// A neutral working directory, so that no project memory or settings of the repository are read by the child.
	dir := filepath.Join(os.TempDir(), "moly-claude-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "claude", "-p", "--system-prompt", system, "--tools", "", "--no-session-persistence", "--model", c.Model)
	cmd.Dir = dir
	cmd.Env = claudeCLIEnv(os.Environ())
	cmd.Stdin = strings.NewReader(req.UserPrompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	start := time.Now()
	if err := cmd.Run(); err != nil {
		// stderr can carry the CLI's own diagnostics, never the prompt; it is logged cut short.
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("claude cli failed: %v %s", err, msg)
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return nil, errors.New("claude cli returned nothing")
	}
	log.Printf("[ClaudeCLI] answered in %.1fs", time.Since(start).Seconds())
	return &LLMResponse{Content: out, Model: c.Model, ProcessingTimeMs: time.Since(start).Milliseconds()}, nil
}
