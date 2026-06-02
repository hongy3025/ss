package parser

import "testing"

func TestHostEntry_Display(t *testing.T) {
	tests := []struct {
		name  string
		entry HostEntry
		want  string
	}{
		{
			name:  "all fields populated",
			entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", User: "root", Port: "22"},
			want:  "dev → root@10.0.0.1:22",
		},
		{
			name:  "missing port",
			entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", User: "root"},
			want:  "dev → root@10.0.0.1",
		},
		{
			name:  "missing user",
			entry: HostEntry{Alias: "dev", HostName: "10.0.0.1", Port: "2222"},
			want:  "dev → 10.0.0.1:2222",
		},
		{
			name:  "missing host falls back to alias",
			entry: HostEntry{Alias: "dev", User: "root"},
			want:  "dev → root@dev",
		},
		{
			name:  "only alias",
			entry: HostEntry{Alias: "dev"},
			want:  "dev → dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.Display(); got != tt.want {
				t.Errorf("Display() = %q, want %q", got, tt.want)
			}
		})
	}
}
