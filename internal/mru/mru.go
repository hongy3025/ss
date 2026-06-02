package mru

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/hongy3025/ss/internal/parser"
)

type MRUEntry struct {
	LastUsed time.Time `json:"lastUsed"`
	Count    int       `json:"count"`
}

type MRU struct {
	Path    string              `json:"-"`
	Entries map[string]MRUEntry `json:"entries"`
}

type Store interface {
	Load() error
	Save() error
	Record(alias string)
	SortEntries(entries []parser.HostEntry) []parser.HostEntry
	Clean(validAliases map[string]bool)
}

func New(path string) Store {
	return &MRU{
		Path:    path,
		Entries: make(map[string]MRUEntry),
	}
}

func (m *MRU) Load() error {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded MRU
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil
	}
	m.Entries = loaded.Entries
	return nil
}

func (m *MRU) Save() error {
	dir := filepath.Dir(m.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0644)
}

func (m *MRU) Record(alias string) {
	entry := m.Entries[alias]
	entry.LastUsed = time.Now()
	entry.Count++
	m.Entries[alias] = entry
}

func (m *MRU) SortEntries(entries []parser.HostEntry) []parser.HostEntry {
	// TODO: implement in next task
	return entries
}

func (m *MRU) Clean(validAliases map[string]bool) {
	// TODO: implement in next task
}
