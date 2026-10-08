package main

import (
	"path/filepath"
	"testing"
	"time"

	"moly/database"
	"moly/models"
)

func TestContextSurvivesRestart(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "restart.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)

	before := &APIServer{database: db}
	saved := &PreviousExtraction{
		Entities:    []models.ExtractedEntity{{}, {}, {}},
		Goal:        "find a good restaurant",
		Values:      []string{"honesty"},
		PrimaryGoal: "find a good restaurant",
	}
	before.savePreviousExtraction("u1", "c1", saved)

	after := &APIServer{database: db}
	got := after.loadPreviousExtraction("u1", "c1")
	if got == nil {
		t.Fatal("context not found after restart")
	}
	if got.Goal != saved.Goal || got.PrimaryGoal != saved.PrimaryGoal || len(got.Entities) != 3 || len(got.Values) != 1 {
		t.Fatalf("context changed across restart: %+v", got)
	}
	if other := after.loadPreviousExtraction("u2", "c1"); other != nil {
		t.Fatal("another user must not load this context")
	}
}

func TestTrackerStateSurvivesRestart(t *testing.T) {
	db, err := database.Init(filepath.Join(t.TempDir(), "tracker.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)

	(&APIServer{database: db}).savePreviousExtraction("u1", "c1", &PreviousExtraction{
		Goal:           "g",
		ContextTracker: &models.ContextTrackerState{PreviousIntent: "ask", PreviousGoals: []string{"g"}},
	})
	got := (&APIServer{database: db}).loadPreviousExtraction("u1", "c1")
	if got == nil || got.ContextTracker == nil || got.ContextTracker.PreviousIntent != "ask" {
		t.Fatalf("tracker state not restored: %+v", got)
	}
}
