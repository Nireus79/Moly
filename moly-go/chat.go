package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ChatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model"`
	Tone    string `json:"tone"`
	Mode    string `json:"mode"`
}

type ChatResponse struct {
	Success   bool   `json:"success"`
	Response  string `json:"response"`
	Error     string `json:"error"`
	Provider  string `json:"provider"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Message == "" {
		respondError(w, http.StatusBadRequest, "Message required")
		return
	}

	config := loadConfig()

	// Route to appropriate provider
	var response string
	var provider string
	var err error

	switch config.Provider {
	case "local", "ollama":
		response, err = chatWithOllama(req.Message, req.Model)
		provider = "ollama"
	case "claude":
		response, err = chatWithClaude(req.Message, req.Model)
		provider = "claude"
	case "openai":
		response, err = chatWithOpenAI(req.Message, req.Model)
		provider = "openai"
	default:
		response, err = chatWithOllama(req.Message, req.Model)
		provider = "ollama"
	}

	if err != nil {
		respondJSON(w, http.StatusOK, ChatResponse{
			Success:  false,
			Error:    err.Error(),
			Provider: provider,
		})
		return
	}

	respondJSON(w, http.StatusOK, ChatResponse{
		Success:  true,
		Response: response,
		Provider: provider,
	})
}

func chatWithOllama(message string, model string) (string, error) {
	if !ollamaIsRunning() {
		return "", fmt.Errorf("ollama not running")
	}

	if model == "" {
		model = "mistral"
	}

	payload := map[string]interface{}{
		"model":  model,
		"prompt": message,
		"stream": false,
	}

	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Post(
		"http://localhost:11434/api/generate",
		"application/json",
		bytes.NewReader(jsonPayload),
	)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	response, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("invalid ollama response")
	}

	return response, nil
}

func chatWithClaude(message string, model string) (string, error) {
	config := loadConfig()

	apiKey, ok := config.APIKeys["claude"].(string)
	if !ok || apiKey == "" {
		return "", fmt.Errorf("claude API key not configured")
	}

	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	payload := map[string]interface{}{
		"model": model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": message,
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("claude error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("invalid claude response")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid claude response format")
	}

	response, ok := firstContent["text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid claude response text")
	}

	return response, nil
}

func chatWithOpenAI(message string, model string) (string, error) {
	config := loadConfig()

	apiKey, ok := config.APIKeys["openai"].(string)
	if !ok || apiKey == "" {
		return "", fmt.Errorf("openai API key not configured")
	}

	if model == "" {
		model = "gpt-3.5-turbo"
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": message,
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid openai response")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid openai response format")
	}

	message_obj, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid openai message format")
	}

	response, ok := message_obj["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid openai response text")
	}

	return response, nil
}
