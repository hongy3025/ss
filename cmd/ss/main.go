package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hongy3025/ss/internal/connector"
	"github.com/hongy3025/ss/internal/mru"
	"github.com/hongy3025/ss/internal/parser"
	"github.com/hongy3025/ss/internal/selector"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	keepMode := false
	for _, arg := range args {
		if arg == "-k" || arg == "--keep" {
			keepMode = true
		}
	}

	configPath, err := parser.DefaultConfigPath()
	if err != nil {
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}

	mruPath := filepath.Join(home, ".ssh-selector", "mru.json")
	mruStore := mru.New(mruPath)
	if err := mruStore.Load(); err != nil {
		fmt.Fprintln(stderr, "ss: warning: failed to load MRU:", err)
	}

	sel := selector.NewFuzzyFinderProvider()
	lastSelectedAlias := ""

	for {
		entries, err := parser.ParseFile(configPath)
		if err != nil {
			fmt.Fprintln(stderr, "ss:", err)
			return 1
		}
		if len(entries) == 0 {
			fmt.Fprintln(stderr, "no ssh host entries found")
			return 1
		}

		validAliases := make(map[string]bool)
		for _, e := range entries {
			validAliases[e.Alias] = true
		}
		mruStore.Clean(validAliases)

		sortedEntries := mruStore.SortEntries(entries)

		entry, err := sel.Find(sortedEntries, lastSelectedAlias)
		if err != nil {
			if errors.Is(err, selector.ErrAbort) {
				return 0
			}
			fmt.Fprintln(stderr, "ss:", err)
			return 1
		}

		mruStore.Record(entry.Alias)
		if err := mruStore.Save(); err != nil {
			fmt.Fprintln(stderr, "ss: warning: failed to save MRU:", err)
		}

		conn := connector.New()
		if err := conn.Connect(entry); err != nil {
			fmt.Fprintln(stderr, "ss:", err)
			return 1
		}

		if !keepMode {
			return 0
		}

		lastSelectedAlias = entry.Alias
	}
}
