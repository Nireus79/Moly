package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const ollamaURL = "http://localhost:11434"

func checkOllama() (installed bool, running bool) {
	// Check if Ollama is installed
	_, err := exec.LookPath("ollama")
	installed = err == nil

	// Check if Ollama is running
	running = ollamaIsRunning()

	return
}

func ollamaIsRunning() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(ollamaURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func getOllamaModels() ([]interface{}, error) {
	if !ollamaIsRunning() {
		return []interface{}{}, fmt.Errorf("ollama not running")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaURL + "/api/tags")
	if err != nil {
		return []interface{}{}, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []interface{}{}, err
	}

	models, ok := result["models"].([]interface{})
	if !ok {
		return []interface{}{}, nil
	}

	return models, nil
}

func pullOllamaModel(modelName string) error {
	if !ollamaIsRunning() {
		return fmt.Errorf("ollama not running")
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	reqBody := map[string]string{"name": modelName}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", ollamaURL+"/api/pull", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama error: %s", string(body))
	}

	return nil
}

func removeOllamaModel(modelName string) error {
	if !ollamaIsRunning() {
		return fmt.Errorf("ollama not running")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	reqBody := map[string]string{"name": modelName}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("DELETE", ollamaURL+"/api/delete", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama error: %s", string(body))
	}

	return nil
}

func startOllama() error {
	if ollamaIsRunning() {
		return nil // Already running
	}

	// Try to start Ollama
	cmd := exec.Command("ollama", "serve")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama: %v", err)
	}

	// Wait for it to be ready
	for i := 0; i < 30; i++ {
		if ollamaIsRunning() {
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("ollama failed to start within 30 seconds")
}

func stopOllama() error {
	// Try to gracefully stop Ollama
	cmd := exec.Command("pkill", "-f", "ollama serve")
	_ = cmd.Run() // Ignore error, might not be running

	// Wait for it to stop
	for i := 0; i < 10; i++ {
		if !ollamaIsRunning() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Force kill if still running
	exec.Command("pkill", "-9", "ollama").Run()

	return nil
}
