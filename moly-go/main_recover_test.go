package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"moly/models"
)

func TestPreviousEntitiesFirstMessage(t *testing.T) {
	if got := previousEntities(nil); len(got) != 0 {
		t.Fatalf("expected no entities for first message, got %d", len(got))
	}
}

func TestPreviousEntitiesReturnsSaved(t *testing.T) {
	prev := &PreviousExtraction{Entities: []models.ExtractedEntity{{}, {}}}
	if got := previousEntities(prev); len(got) != 2 {
		t.Fatalf("expected 2 saved entities, got %d", len(got))
	}
}

func TestCorsMiddlewareRecoversPanic(t *testing.T) {
	h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/message-processor", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"error\":\"Internal server error\"}\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}
