package tools

import (
	"testing"
	"time"
)

func TestLLMCache_SetGet(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 100)

	cache.Set("test input", "test output", "extraction")

	output, found := cache.Get("test input", "extraction")
	if !found {
		t.Errorf("Expected to find cached entry")
	}

	if output != "test output" {
		t.Errorf("Expected 'test output', got %q", output)
	}
}

func TestLLMCache_MissingKey(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 100)

	output, found := cache.Get("non-existent", "extraction")
	if found {
		t.Errorf("Expected not to find entry for non-existent key")
	}

	if output != "" {
		t.Errorf("Expected empty output, got %q", output)
	}
}

func TestLLMCache_TypeSeparation(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 100)

	cache.Set("input", "extraction_result", "extraction")
	cache.Set("input", "intent_result", "intent")

	extraction, _ := cache.Get("input", "extraction")
	intent, _ := cache.Get("input", "intent")

	if extraction != "extraction_result" {
		t.Errorf("Expected 'extraction_result', got %q", extraction)
	}

	if intent != "intent_result" {
		t.Errorf("Expected 'intent_result', got %q", intent)
	}
}

func TestLLMCache_Expiration(t *testing.T) {
	cache := NewLLMCache(100*time.Millisecond, 100) // Very short TTL

	cache.Set("input", "output", "extraction")

	// Should be in cache immediately
	_, found := cache.Get("input", "extraction")
	if !found {
		t.Errorf("Expected to find entry immediately after Set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, found = cache.Get("input", "extraction")
	if found {
		t.Errorf("Expected entry to expire after TTL")
	}
}


func TestLLMCache_Size(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 100)

	if cache.Size() != 0 {
		t.Errorf("Expected initial size to be 0, got %d", cache.Size())
	}

	cache.Set("input1", "output1", "extraction")
	if cache.Size() != 1 {
		t.Errorf("Expected size 1 after one Set, got %d", cache.Size())
	}

	cache.Set("input2", "output2", "extraction")
	if cache.Size() != 2 {
		t.Errorf("Expected size 2 after two Sets, got %d", cache.Size())
	}

	// Setting same input/type overwrites
	cache.Set("input1", "new output", "extraction")
	if cache.Size() != 2 {
		t.Errorf("Expected size to remain 2 after overwrite, got %d", cache.Size())
	}
}




func TestLLMCache_DefaultCache(t *testing.T) {
	cache := NewDefaultLLMCache()

	cache.Set("input", "output", "extraction")

	output, found := cache.Get("input", "extraction")
	if !found {
		t.Errorf("Expected to find entry in default cache")
	}

	if output != "output" {
		t.Errorf("Expected 'output', got %q", output)
	}
}

func TestLLMCache_LargeNumberOfEntries(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 1000)

	// Add many entries
	for i := 0; i < 500; i++ {
		cache.Set("input"+string(rune(i)), "output"+string(rune(i)), "extraction")
	}

	if cache.Size() != 500 {
		t.Errorf("Expected 500 entries, got %d", cache.Size())
	}

	// All should be retrievable
	for i := 0; i < 500; i++ {
		output, found := cache.Get("input"+string(rune(i)), "extraction")
		if !found {
			t.Errorf("Expected to find entry %d", i)
			break
		}
		if output != "output"+string(rune(i)) {
			t.Errorf("Entry %d has wrong output", i)
		}
	}
}

func TestLLMCache_EvictionOnMaxSize(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 10) // Small max size

	// Add 20 entries (should trigger eviction)
	for i := 0; i < 20; i++ {
		cache.Set("input"+string(rune(i)), "output"+string(rune(i)), "extraction")
	}

	// Size should be capped near maxSize
	if cache.Size() > 15 {
		t.Errorf("Expected size to be capped, got %d", cache.Size())
	}
}

func TestLLMCache_Concurrent(t *testing.T) {
	cache := NewLLMCache(1*time.Hour, 1000)

	// Simulate concurrent access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Set("input"+string(rune(id*100+j)), "output"+string(rune(id*100+j)), "extraction")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have entries from all goroutines
	if cache.Size() < 100 {
		t.Errorf("Expected significant number of entries from concurrent access, got %d", cache.Size())
	}
}

