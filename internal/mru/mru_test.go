package mru

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)
	if store == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)
	if err := store.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	mru := store.(*MRU)
	if len(mru.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(mru.Entries))
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	store.Record("dev")

	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	store2 := New(path)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	mru := store2.(*MRU)
	entry, ok := mru.Entries["dev"]
	if !ok {
		t.Fatal("Entries[dev] not found")
	}
	if entry.Count != 1 {
		t.Errorf("Count = %d, want 1", entry.Count)
	}
	if time.Since(entry.LastUsed) > time.Minute {
		t.Errorf("LastUsed is too old: %v", entry.LastUsed)
	}
}
