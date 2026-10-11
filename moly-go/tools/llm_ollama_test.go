package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ollama ignores a top-level "temperature" (checked against a live mistral: four runs at 0 gave four different sentences, and
// identical ones with options.temperature), so the request must carry it under "options".
func TestOllamaRequestCarriesTemperatureInOptionsAndAsksForJSON(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"response":"{}","model":"m"}`))
	}))
	defer srv.Close()
	c := &LLMClient{Provider: "ollama", Model: "m", Temperature: 0.2, OllamaEndpoint: srv.URL}
	if _, err := c.callOllama(context.Background(), &LLMRequest{UserPrompt: "x", Temperature: 0.3, JSONMode: true}); err != nil {
		t.Fatal(err)
	}
	if _, top := got["temperature"]; top {
		t.Fatal("temperature must not be sent at the top level")
	}
	opts, _ := got["options"].(map[string]interface{})
	if opts["temperature"] != 0.3 {
		t.Fatalf("options.temperature = %v, want 0.3", opts["temperature"])
	}
	if got["format"] != "json" {
		t.Fatalf("format = %v, want json", got["format"])
	}
}
