package parser

import (
	"os"
	"path/filepath"
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

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}
