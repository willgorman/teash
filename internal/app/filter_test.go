package app

import (
	"testing"

	"github.com/willgorman/teash/internal/teleport"
)

func serverView(hostname, addr, os string, labels map[string]string) ServerView {
	return ServerView{
		Server: teleport.Server{
			Hostname: hostname,
			Addr:     addr,
			OS:       os,
		},
		AllLabels: labels,
	}
}

func TestFilterServersByText(t *testing.T) {
	servers := []ServerView{
		serverView("web1", "10.0.0.1", "Rocky Linux 9", map[string]string{"environment": "dev"}),
		serverView("db1", "10.0.0.2", "Ubuntu 22.04", map[string]string{"environment": "production"}),
	}

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
			got := FilterServersByText(servers, tt.query)
			var gotHosts []string
			for _, sv := range got {
				gotHosts = append(gotHosts, sv.Hostname)
			}
			if len(gotHosts) != len(tt.wantHosts) {
				t.Fatalf("FilterServersByText(%q) = %v, want %v", tt.query, gotHosts, tt.wantHosts)
			}
			for i := range gotHosts {
				if gotHosts[i] != tt.wantHosts[i] {
					t.Fatalf("FilterServersByText(%q) = %v, want %v", tt.query, gotHosts, tt.wantHosts)
				}
			}
		})
	}
}

func TestFilterServers(t *testing.T) {
	servers := []ServerView{
		serverView("web1", "10.0.0.1", "linux", map[string]string{"role": "web"}),
		serverView("web2", "10.0.0.2", "darwin", map[string]string{"role": "web"}),
		serverView("db1", "10.0.0.3", "linux", map[string]string{"role": "database"}),
	}

	tests := []struct {
		name       string
		colFilters []ColumnFilter
		wantHosts  []string
	}{
		{
			name:       "no filters returns all servers",
			colFilters: nil,
			wantHosts:  []string{"web1", "web2", "db1"},
		},
		{
			name: "single column, single value",
			colFilters: []ColumnFilter{
				{Column: ColOS, Values: []string{"linux"}},
			},
			wantHosts: []string{"web1", "db1"},
		},
		{
			name: "single column, multiple values ORed together",
			colFilters: []ColumnFilter{
				{Column: ColOS, Values: []string{"linux", "darwin"}},
			},
			wantHosts: []string{"web1", "web2", "db1"},
		},
		{
			name: "multiple columns ANDed together",
			colFilters: []ColumnFilter{
				{Column: ColOS, Values: []string{"linux", "darwin"}},
				{Column: "role", Values: []string{"web"}},
			},
			wantHosts: []string{"web1", "web2"},
		},
		{
			name: "column with empty Values is ignored",
			colFilters: []ColumnFilter{
				{Column: ColOS, Values: nil},
				{Column: "role", Values: []string{"database"}},
			},
			wantHosts: []string{"db1"},
		},
		{
			name: "no matches returns empty",
			colFilters: []ColumnFilter{
				{Column: ColOS, Values: []string{"windows"}},
			},
			wantHosts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterServers(servers, tt.colFilters)
			var gotHosts []string
			for _, sv := range got {
				gotHosts = append(gotHosts, sv.Hostname)
			}
			if len(gotHosts) != len(tt.wantHosts) {
				t.Fatalf("FilterServers() = %v, want %v", gotHosts, tt.wantHosts)
			}
			for i := range gotHosts {
				if gotHosts[i] != tt.wantHosts[i] {
					t.Fatalf("FilterServers() = %v, want %v", gotHosts, tt.wantHosts)
				}
			}
		})
	}
}

func TestDistinctColumnValues(t *testing.T) {
	servers := []ServerView{
		serverView("web1", "10.0.0.1", "linux", map[string]string{"role": "web"}),
		serverView("web2", "10.0.0.2", "darwin", map[string]string{"role": "web"}),
		serverView("db1", "10.0.0.3", "linux", map[string]string{}),
	}

	got := DistinctColumnValues(servers, ColOS)
	want := []string{"darwin", "linux"}
	if len(got) != len(want) {
		t.Fatalf("DistinctColumnValues(OS) = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("DistinctColumnValues(OS) = %v, want %v", got, want)
		}
	}

	gotRole := DistinctColumnValues(servers, "role")
	wantRole := []string{"web"}
	if len(gotRole) != len(wantRole) || gotRole[0] != wantRole[0] {
		t.Fatalf("DistinctColumnValues(role) = %v, want %v (empty label value from db1 must be excluded)", gotRole, wantRole)
	}
}
