//go:build !windows

package connector

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/hongy3025/ss/internal/parser"
)

func newPlatformConnector() Connector {
	return &UnixConnector{}
}

type UnixConnector struct{}

func (c *UnixConnector) Connect(entry parser.HostEntry) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}
	return syscall.Exec(sshPath, []string{"ssh", entry.Alias}, os.Environ())
}
