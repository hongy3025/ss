package parser

import (
	"strings"
	"testing"
)

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("Z:/this/path/does/not/exist/config")
	if err == nil {
		t.Fatal("ParseFile() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ssh config not found") {
		t.Errorf("error %q should mention 'ssh config not found'", err)
	}
}
