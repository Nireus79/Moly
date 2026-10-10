package database

import (
	"path/filepath"
	"testing"
	"time"

	"moly/models"
)

func TestGetByUserIDReturnsContactsWithAllColumns(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "contacts.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)

	repo := NewContactRepository(db)
	contact := &models.Contact{UserID: "u1", Name: "Ann", Relationship: "friend", Status: "active", CreatedVia: "conversation"}
	if err := repo.Save(contact); err != nil {
		t.Fatalf("save: %v", err)
	}

	list, err := repo.GetByUserID("u1")
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Ann" {
		t.Fatalf("expected Ann back, got %+v", list)
	}
}
