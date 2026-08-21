package tui

import (
	"testing"

	"github.com/willgorman/teash/internal/app"
	"github.com/willgorman/teash/internal/teleport"
)

func serverView(hostname, addr, os string, labels map[string]string) app.ServerView {
	return app.ServerView{
		Server: teleport.Server{
			Hostname: hostname,
			Addr:     addr,
			OS:       os,
		},
		AllLabels: labels,
	}
}

func TestFilterServersByText(t *testing.T) {
	servers := []app.ServerView{
		serverView("web1", "10.0.0.1", "Rocky Linux 9", map[string]string{"environment": "dev"}),
		serverView("db1", "10.0.0.2", "Ubuntu 22.04", map[string]string{"environment": "production"}),
	}
	m := Model{servers: servers}

	tests := []struct {
		name      string
		query     string
		wantHosts []string
	}{
		{
			name:      "multi-word query matches across OS and label columns",
			query:     "rocky dev",
			wantHosts: []string{"web1"},
		},
		{
			name:      "fuzzy match skips characters within a single column",
			query:     "Rocky 9",
			wantHosts: []string{"web1"},
		},
		{
			name:      "typo-tolerant fuzzy match",
			query:     "rcky",
			wantHosts: []string{"web1"},
		},
		{
			name:      "existing substring-style single word query still matches",
			query:     "web",
			wantHosts: []string{"web1"},
		},
		{
			name:      "word with no match anywhere excludes the server",
			query:     "rocky nonexistentword",
			wantHosts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.filterServersByText(tt.query)
			var gotHosts []string
			for _, sv := range got {
				gotHosts = append(gotHosts, sv.Hostname)
			}
			if len(gotHosts) != len(tt.wantHosts) {
				t.Fatalf("filterServersByText(%q) = %v, want %v", tt.query, gotHosts, tt.wantHosts)
			}
			for i := range gotHosts {
				if gotHosts[i] != tt.wantHosts[i] {
					t.Fatalf("filterServersByText(%q) = %v, want %v", tt.query, gotHosts, tt.wantHosts)
				}
			}
		})
	}
}
