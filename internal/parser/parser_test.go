package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostEntry_Display(t *testing.T) {
	tests := []struct {
		name  string
		entry HostEntry
		want  string
	}{
		{
			name:  "all fields populated",
			entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", User: "root", Port: "22"},
			want:  "dev → root@10.0.0.1:22",
		},
		{
			name:  "missing port",
			entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", User: "root"},
			want:  "dev → root@10.0.0.1",
		},
		{
			name:  "missing user",
			entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", Port: "2222"},
			want:  "dev → 10.0.0.1:2222",
		},
		{
			name:  "missing host falls back to alias",
			entry: HostEntry{Alias: "dev", User: "root"},
			want:  "dev → root@dev",
		},
		{
			name:  "only alias",
			entry: HostEntry{Alias: "dev"},
			want:  "dev → dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.Display(); got != tt.want {
				t.Errorf("Display() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse_BasicHostBlock(t *testing.T) {
	input := `Host dev
    HostName 10.0.0.1
    User root
    Port 22
    IdentityFile ~/.ssh/id_ed25519
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1", len(entries))
	}
	got := entries[0]
	want := HostEntry{
		Alias:        "dev",
		HostName:     "10.0.0.1",
		User:         "root",
		Port:         "22",
		IdentityFile: "~/.ssh/id_ed25519",
	}
	if got != want {
		t.Errorf("Parse() entry = %+v, want %+v", got, want)
	}
}

func TestParse_MultipleHosts(t *testing.T) {
	input := `Host dev
    HostName 10.0.0.1
    User root

Host prod
    HostName prod.example.com
    User deploy
    Port 2222
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}
	if entries[0].Alias != "dev" || entries[0].HostName != "10.0.0.1" {
		t.Errorf("entries[0] = %+v", entries[0])
	}
	if entries[1].Alias != "prod" || entries[1].Port != "2222" {
		t.Errorf("entries[1] = %+v", entries[1])
	}
}

func TestParse_CommentsAndBlankLines(t *testing.T) {
	input := `# this is a comment
# another comment

Host dev
    # inline comment in block
    HostName 10.0.0.1

    User root
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1", len(entries))
	}
	if entries[0].User != "root" {
		t.Errorf("User = %q, want root", entries[0].User)
	}
}

func TestParse_SkipWildcardHosts(t *testing.T) {
	input := `Host *
    User root

Host dev
    HostName 10.0.0.1

Host *.example.com
    User deploy
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1", len(entries))
	}
	if entries[0].Alias != "dev" {
		t.Errorf("Alias = %q, want dev", entries[0].Alias)
	}
}

func TestParse_MultiplePatterns(t *testing.T) {
	input := `Host dev prod
    HostName 10.0.0.1
    User root
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Parse() returned %d entries, want 2", len(entries))
	}
	if entries[0].Alias != "dev" {
		t.Errorf("entries[0].Alias = %q, want dev", entries[0].Alias)
	}
	if entries[1].Alias != "prod" {
		t.Errorf("entries[1].Alias = %q, want prod", entries[1].Alias)
	}
	if entries[0].HostName != "10.0.0.1" || entries[1].HostName != "10.0.0.1" {
		t.Errorf("HostName not shared: %+v", entries)
	}
	if entries[0].User != "root" || entries[1].User != "root" {
		t.Errorf("User not shared: %+v", entries)
	}
}

func TestParse_MultiplePatternsWithWildcard(t *testing.T) {
	input := `Host dev * prod staging
    HostName 10.0.0.1
`
	entries, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Parse() returned %d entries, want 3", len(entries))
	}
	wantAliases := []string{"dev", "prod", "staging"}
	for i, want := range wantAliases {
		if entries[i].Alias != want {
			t.Errorf("entries[%d].Alias = %q, want %q", i, entries[i].Alias, want)
		}
	}
}

func TestDefaultConfigPath(t *testing.T) {
	got, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Fatalf("os.UserHomeDir() error = %v", homeErr)
	}
	want := filepath.Join(home, ".ssh", "config")
	if got != want {
		t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
	}
}
