package mru

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hongy3025/ss/internal/parser"
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

func TestRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	// 第一次记录
	store.Record("dev")
	mru := store.(*MRU)
	entry := mru.Entries["dev"]
	if entry.Count != 1 {
		t.Errorf("Count = %d, want 1", entry.Count)
	}

	// 第二次记录
	store.Record("dev")
	entry = mru.Entries["dev"]
	if entry.Count != 2 {
		t.Errorf("Count = %d, want 2", entry.Count)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	if err := os.WriteFile(path, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	store := New(path)
	if err := store.Load(); err == nil {
		t.Fatal("Load() expected error for invalid JSON, got nil")
	}
}

func TestSortEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mru.json")
	store := New(path)

	// 记录使用顺序：prod 先用，dev 后用
	store.Record("prod")
	time.Sleep(10 * time.Millisecond)
	store.Record("dev")

	entries := []parser.HostEntry{
		{Alias: "prod"},
		{Alias: "staging"},
		{Alias: "dev"},
	}

	sorted := store.SortEntries(entries)

	// 最近使用的应该排最前
	if sorted[0].Alias != "dev" {
		t.Errorf("sorted[0].Alias = %q, want dev", sorted[0].Alias)
	}
	if sorted[1].Alias != "prod" {
		t.Errorf("sorted[1].Alias = %q, want prod", sorted[1].Alias)
	}
	if sorted[2].Alias != "staging" {
		t.Errorf("sorted[2].Alias = %q, want staging", sorted[2].Alias)
	}
}
