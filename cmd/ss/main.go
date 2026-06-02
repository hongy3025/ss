package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/hongy3025/ss/internal/connector"
	"github.com/hongy3025/ss/internal/parser"
	"github.com/hongy3025/ss/internal/selector"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	configPath, err := parser.DefaultConfigPath()
	if err != nil {
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}

	entries, err := parser.ParseFile(configPath)
	if err != nil {
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}
	if len(entries) == 0 {
		fmt.Fprintln(stderr, "no ssh host entries found")
		return 1
	}

	sel := selector.NewFuzzyFinderProvider()
	entry, err := sel.Find(entries)
	if err != nil {
		if errors.Is(err, selector.ErrAbort) {
			return 0
		}
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}

	conn := connector.New()
	if err := conn.Connect(entry); err != nil {
		fmt.Fprintln(stderr, "ss:", err)
		return 1
	}
	return 0
}
