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
