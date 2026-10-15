package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// What models exist is asked of the provider, with the user's own key, never assumed: a list written into the program is
// out of date the day a provider adds or retires a model. This file is the only place that asks (besides the chat calls in
// llm_client.go), so the vendors' addresses stay in these two files.

// ModelInfo is one model a provider offers.
type ModelInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Line  string `json:"line,omitempty"` // Claude: haiku, sonnet, opus, ... (as the provider says)
}

// ModelList is what a provider offers, and the one to preselect when the user has not chosen.
type ModelList struct {
	Models    []ModelInfo `json:"models"`
	Suggested string      `json:"suggested"`
}

var (
	claudeModelsURL = "https://api.anthropic.com/v1/models"
	openAIModelsURL = "https://api.openai.com/v1/models"
)

// openAINotChat marks the OpenAI models that are not for chat. The list of models has no field that says what a model is for,
// so these families (embeddings, speech, images, moderation, completions-only models) are left out; any model the user
// wants that is left out can still be typed in.
var openAINotChat = regexp.MustCompile(`(?i)(embedding|whisper|tts|dall-e|moderation|image|audio|realtime|transcribe|search|davinci|babbage|instruct|computer-use)`)

// ListModels asks the provider which models the settings' key can use. For Ollama no key is needed: the server's own list.
func ListModels(ctx context.Context, s LLMSettings) (*ModelList, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	switch strings.ToLower(strings.TrimSpace(s.Provider)) {
	case "claude":
		return listClaudeModels(ctx, s.APIKey)
	case "openai":
		return listOpenAIModels(ctx, s.APIKey)
	case "ollama":
		endpoint := s.OllamaEndpoint
		if endpoint == "" {
			endpoint = "http://127.0.0.1:11434"
		}
		return listOllamaModels(ctx, endpoint)
	}
	return nil, fmt.Errorf("unknown AI provider %q", s.Provider)
}

func getJSON(ctx context.Context, rawURL string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("the provider did not answer: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return fmt.Errorf("the provider answered %d: %s", resp.StatusCode, msg)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("the provider's list could not be read: %w", err)
	}
	return nil
}

// listClaudeModels reads the models the key can use that are active (not retired). The provider lists the newest first.
// The suggestion is the newest model of the "sonnet" line (the balanced one: the newest overall is usually the dearest),
// or else the first.
func listClaudeModels(ctx context.Context, key string) (*ModelList, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("an API key is needed to list the models")
	}
	list := &ModelList{Models: []ModelInfo{}}
	after := ""
	for page := 0; page < 5; page++ {
		// No server-side filter: its parameter name was refused twice (found live, 2026-10-14). Retired models are left out below,
		// from the lifecycle each model reports.
		q := url.Values{"limit": {"1000"}}
		if after != "" {
			q.Set("after_id", after)
		}
		var resp struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
				Line        string `json:"line"`
				Lifecycle   string `json:"lifecycle"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := getJSON(ctx, claudeModelsURL+"?"+q.Encode(), map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"}, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Data {
			if m.ID == "" || (m.Lifecycle != "" && m.Lifecycle != "active") {
				continue
			}
			label := m.DisplayName
			if label == "" {
				label = m.ID
			}
			list.Models = append(list.Models, ModelInfo{ID: m.ID, Label: label, Line: m.Line})
		}
		if !resp.HasMore || resp.LastID == "" {
			break
		}
		after = resp.LastID
	}
	for _, m := range list.Models {
		if m.Line == "sonnet" {
			list.Suggested = m.ID
			break
		}
	}
	if list.Suggested == "" && len(list.Models) > 0 {
		list.Suggested = list.Models[0].ID
	}
	return list, nil
}

// listOpenAIModels reads the models of the key, leaves out those that are not for chat, and lists the newest first.
func listOpenAIModels(ctx context.Context, key string) (*ModelList, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("an API key is needed to list the models")
	}
	var resp struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		} `json:"data"`
	}
	if err := getJSON(ctx, openAIModelsURL, map[string]string{"Authorization": "Bearer " + key}, &resp); err != nil {
		return nil, err
	}
	type entry struct {
		id      string
		created int64
	}
	var entries []entry
	for _, m := range resp.Data {
		if m.ID != "" && !openAINotChat.MatchString(m.ID) {
			entries = append(entries, entry{m.ID, m.Created})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].created > entries[j].created })
	list := &ModelList{Models: []ModelInfo{}}
	for _, e := range entries {
		list.Models = append(list.Models, ModelInfo{ID: e.id, Label: e.id})
	}
	if len(list.Models) > 0 {
		list.Suggested = list.Models[0].ID
	}
	return list, nil
}

// listOllamaModels reads the models the Ollama server has. The first is suggested.
func listOllamaModels(ctx context.Context, endpoint string) (*ModelList, error) {
	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := getJSON(ctx, strings.TrimRight(endpoint, "/")+"/api/tags", nil, &resp); err != nil {
		return nil, fmt.Errorf("Ollama did not answer at %s", endpoint)
	}
	list := &ModelList{Models: []ModelInfo{}}
	for _, m := range resp.Models {
		if m.Name != "" {
			list.Models = append(list.Models, ModelInfo{ID: m.Name, Label: m.Name})
		}
	}
	if len(list.Models) > 0 {
		list.Suggested = list.Models[0].ID
	}
	return list, nil
}

// detectModel is the model to use when none was chosen: the provider's own suggestion. Nothing is assumed.
func detectModel(provider, apiKey, ollamaEndpoint string) (string, error) {
	list, err := ListModels(context.Background(), LLMSettings{Provider: provider, APIKey: apiKey, OllamaEndpoint: ollamaEndpoint})
	if err != nil {
		return "", fmt.Errorf("no model is chosen and the available models could not be listed: %w", err)
	}
	if list.Suggested == "" {
		return "", fmt.Errorf("no model is chosen and the provider lists none for this key")
	}
	return list.Suggested, nil
}
