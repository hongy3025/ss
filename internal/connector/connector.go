package connector

import "github.com/hongy3025/ss/internal/parser"

type Connector interface {
	Connect(entry parser.HostEntry) error
}

func New() Connector {
	return newPlatformConnector()
}

func buildCommand(entry parser.HostEntry) []string {
	return []string{"ssh", entry.Alias}
}
