package connector

import (
	"reflect"
	"testing"

	"github.com/hongy3025/ss/internal/parser"
)

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name  string
		entry parser.HostEntry
		want  []string
	}{
		{
			name:  "simple alias",
			entry: parser.HostEntry{Alias: "dev"},
			want:  []string{"ssh", "dev"},
		},
		{
			name:  "alias with special chars",
			entry: parser.HostEntry{Alias: "prod-us-east"},
			want:  []string{"ssh", "prod-us-east"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCommand(tt.entry)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
