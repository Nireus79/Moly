package tools

import (
	"moly/models"
	"testing"
	"time"
)

func TestExtractionCacheNewCache(t *testing.T) {
	cache := NewExtractionCache()

	if cache == nil {
		t.Fatal("NewExtractionCache returned nil")
	}

	if cache.Size() != 0 {
		t.Error("New cache should be empty")
	}
}

func TestExtractionCacheSetAndGet(t *testing.T) {
	cache := NewExtractionCache()

	entities := []models.ExtractedEntity{
		{
			Value:      "test",
			Type:       "contact",
			Confidence: 0.95,
		},
	}

	// Set
	cache.Set("user1", "msg1", entities)

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Get
	retrieved, found := cache.Get("user1", "msg1")
	if !found {
		t.Error("Expected to find cached entry")
	}

	if len(retrieved) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(retrieved))
	}

	if retrieved[0].Value != "test" {
		t.Errorf("Expected entity value 'test', got '%s'", retrieved[0].Value)
	}
}

func TestExtractionCacheMiss(t *testing.T) {
	cache := NewExtractionCache()

	_, found := cache.Get("user1", "nonexistent")
	if found {
		t.Error("Should not find nonexistent entry")
	}
}

func TestExtractionCacheDifferentUsers(t *testing.T) {
	cache := NewExtractionCache()

	entities1 := []models.ExtractedEntity{{Value: "user1_entity", Type: "contact"}}
	entities2 := []models.ExtractedEntity{{Value: "user2_entity", Type: "contact"}}

	cache.Set("user1", "msg1", entities1)
	cache.Set("user2", "msg1", entities2)

	retrieved1, _ := cache.Get("user1", "msg1")
	retrieved2, _ := cache.Get("user2", "msg1")

	if retrieved1[0].Value != "user1_entity" {
		t.Error("User1 entity mismatch")
	}

	if retrieved2[0].Value != "user2_entity" {
		t.Error("User2 entity mismatch")
	}
}

func TestExtractionCacheDifferentMessages(t *testing.T) {
	cache := NewExtractionCache()

	entities1 := []models.ExtractedEntity{{Value: "msg1_entity", Type: "contact"}}
	entities2 := []models.ExtractedEntity{{Value: "msg2_entity", Type: "contact"}}

	cache.Set("user1", "msg1", entities1)
	cache.Set("user1", "msg2", entities2)

	retrieved1, _ := cache.Get("user1", "msg1")
	retrieved2, _ := cache.Get("user1", "msg2")

	if retrieved1[0].Value != "msg1_entity" {
		t.Error("Msg1 entity mismatch")
	}

	if retrieved2[0].Value != "msg2_entity" {
		t.Error("Msg2 entity mismatch")
	}
}




func TestExtractionCacheGenerateKey(t *testing.T) {
	cache := NewExtractionCache()

	key1 := cache.generateKey("user1", "msg1")
	key2 := cache.generateKey("user1", "msg1")
	key3 := cache.generateKey("user1", "msg2")

	if key1 != key2 {
		t.Error("Same user+msg should generate same key")
	}

	if key1 == key3 {
		t.Error("Different messages should generate different keys")
	}
}

func TestExtractionCacheConcurrent(t *testing.T) {
	cache := NewExtractionCache()

	entities := []models.ExtractedEntity{{Value: "test", Type: "contact"}}

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			cache.Set("user1", "msg1", entities)
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("user1", "msg1")
		}
	}()

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)

	// Should not panic and should have data
	if cache.Size() == 0 {
		t.Error("Cache should have entries after concurrent operations")
	}
}

func TestExtractionCacheMaxSize(t *testing.T) {
	// This test would require setting a very low maxSize to test eviction
	// For now, just verify the mechanism exists
	cache := NewExtractionCache()

	if cache.maxSize <= 0 {
		t.Error("maxSize should be positive")
	}
}
