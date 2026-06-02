package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type HostEntry struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
}

func (h HostEntry) Display() string {
	host := h.HostName
	if host == "" {
		host = h.Alias
	}
	target := host
	if h.User != "" {
		target = h.User + "@" + host
	}
	if h.Port != "" {
		target = target + ":" + h.Port
	}
	return h.Alias + " → " + target
}

// DefaultConfigPath returns the default OpenSSH client configuration path,
// $HOME/.ssh/config, on the current platform. It returns an error if the
// user's home directory cannot be determined.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// Parse parses SSH config content and returns all valid HostEntry blocks.
func Parse(content string) ([]HostEntry, error) {
	var entries []HostEntry
	var current *HostEntry

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := fields[0]
		value := strings.Join(fields[1:], " ")

		switch key {
		case "Host":
			if current != nil {
				entries = append(entries, *current)
			}
			alias := value
			if strings.ContainsAny(alias, "*?") {
				current = nil
				continue
			}
			current = &HostEntry{Alias: alias}
		default:
			if current == nil {
				continue
			}
			switch key {
			case "HostName":
				current.HostName = value
			case "User":
				current.User = value
			case "Port":
				current.Port = value
			case "IdentityFile":
				current.IdentityFile = value
			}
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
