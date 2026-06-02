//go:build windows

package connector

import (
	"os/exec"

	"github.com/hongy3025/ss/internal/parser"
)

func newPlatformConnector() Connector {
	return &WindowsConnector{}
}

type WindowsConnector struct{}

func (c *WindowsConnector) Connect(entry parser.HostEntry) error {
	if path, err := exec.LookPath("wt.exe"); err == nil {
		return exec.Command(path, "-d", ".", "ssh", entry.Alias).Start()
	}
	return exec.Command(
		"cmd.exe", "/c", "start", "cmd.exe", "/k", "ssh", entry.Alias,
	).Start()
}
