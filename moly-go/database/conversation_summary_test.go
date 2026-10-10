package database

import (
	"path/filepath"
	"testing"
	"time"

	"moly/models"
)

func newSummaryFixture(t *testing.T) (*ConversationSummaryRepository, string) {
	t.Helper()
	db, err := initWithKey(filepath.Join(t.TempDir(), "summary.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	repo := NewConversationSummaryRepository(c)
	if err := repo.CreateSummary(&models.ConversationSummary{
		UserID: "u1", ConversationID: "c1", Arc: "start", SummaryVersion: 1, Confidence: 0.5,
		LastUpdated: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	return repo, "c1"
}

func TestEmptyAccumulatedFieldsRoundTrip(t *testing.T) {
	repo, _ := newSummaryFixture(t)
	got, err := repo.GetSummary("u1", "c1")
	if err != nil || got == nil {
		t.Fatalf("get: %v", err)
	}
	if got.AccumulatedValues != "[]" || got.AccumulatedCharacteristics != "[]" || got.ClarityProgression != "[]" {
		t.Fatalf("empty accumulated fields not stored as valid JSON: %q %q %q",
			got.AccumulatedValues, got.AccumulatedCharacteristics, got.ClarityProgression)
	}
}

func TestStaleWriterGetsVersionConflict(t *testing.T) {
	repo, _ := newSummaryFixture(t)
	stale, _ := repo.GetSummary("u1", "c1")
	fresh, _ := repo.GetSummary("u1", "c1")
	fresh.SummaryVersion++
	fresh.Arc = "fresh"
	if err := repo.UpdateSummary(fresh); err != nil {
		t.Fatalf("first writer: %v", err)
	}
	stale.SummaryVersion++
	if err := repo.UpdateSummary(stale); err != ErrSummaryVersionConflict {
		t.Fatalf("stale writer should conflict, got %v", err)
	}
}

func TestTwoWritersBothSave(t *testing.T) {
	repo, _ := newSummaryFixture(t)
	if err := repo.UpdateSummaryWithRetry("u1", "c1", func(s *models.ConversationSummary) {
		s.Arc = "regenerated arc"
	}); err != nil {
		t.Fatalf("writer 1: %v", err)
	}
	if err := repo.UpdateSummaryWithRetry("u1", "c1", func(s *models.ConversationSummary) {
		s.AccumulatedEntityCount = 7
	}); err != nil {
		t.Fatalf("writer 2: %v", err)
	}
	got, _ := repo.GetSummary("u1", "c1")
	if got.Arc != "regenerated arc" || got.AccumulatedEntityCount != 7 || got.SummaryVersion != 3 {
		t.Fatalf("both changes should be stored: arc=%q count=%d version=%d", got.Arc, got.AccumulatedEntityCount, got.SummaryVersion)
	}
}

// The message handler saved without advancing the version, which always failed the version check.
func TestSaveWithoutVersionBumpAlwaysConflicts(t *testing.T) {
	repo, _ := newSummaryFixture(t)
	s, _ := repo.GetSummary("u1", "c1")
	if err := repo.UpdateSummary(s); err != ErrSummaryVersionConflict {
		t.Fatalf("expected conflict without a version bump, got %v", err)
	}
}
