package parser

import (
	"fmt"
	"os"
)

func ParseFile(path string) ([]HostEntry, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("ssh config not found at %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(content))
}
