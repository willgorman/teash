package main

import (
	"os"
	"testing"
)

func Test_parseNodesJSON(t *testing.T) {
	t.Run("parses lab fixture with correct count and fields", func(t *testing.T) {
		data, err := os.ReadFile("testdata/lab-tsh-ls.json")
		if err != nil {
			t.Fatalf("failed to read fixture: %v", err)
		}
		nodes, err := parseNodesJSON(data)
		if err != nil {
			t.Fatalf("parseNodesJSON() error = %v", err)
		}
		if len(nodes) != 308 {
			t.Errorf("expected 308 nodes, got %d", len(nodes))
		}
		// Verify first node fields
		first := nodes[0]
		if first.Hostname != "server1" {
			t.Errorf("first hostname = %q, want %q", first.Hostname, "ams101-0112-h01-lab")
		}
		if first.OS != "Rocky Linux 9.5 (Blue Onyx)" {
			t.Errorf("first OS = %q, want %q", first.OS, "Rocky Linux 9.5 (Blue Onyx)")
		}
		// Verify labels are populated
		if first.Labels == nil {
			t.Fatal("first node labels should not be nil")
		}
		if first.Labels["env"] != "dev" {
			t.Errorf("first node label env = %q, want %q", first.Labels["env"], "dev")
		}
	})

	t.Run("filters out non-node kinds", func(t *testing.T) {
		input := `[
			{"kind":"node","metadata":{"labels":{}},"spec":{"hostname":"h1","cmd_labels":{"ip":{"result":"1.2.3.4"},"os":{"result":"Linux"}}}},
			{"kind":"app_server","metadata":{"labels":{}},"spec":{"hostname":"a1","cmd_labels":{"ip":{"result":""},"os":{"result":""}}}},
			{"kind":"node","metadata":{"labels":{}},"spec":{"hostname":"h2","cmd_labels":{"ip":{"result":"5.6.7.8"},"os":{"result":"Linux"}}}}
		]`
		nodes, err := parseNodesJSON([]byte(input))
		if err != nil {
			t.Fatalf("parseNodesJSON() error = %v", err)
		}
		if len(nodes) != 2 {
			t.Errorf("expected 2 nodes, got %d", len(nodes))
		}
		if nodes[0].Hostname != "h1" {
			t.Errorf("first hostname = %q, want %q", nodes[0].Hostname, "h1")
		}
		if nodes[1].Hostname != "h2" {
			t.Errorf("second hostname = %q, want %q", nodes[1].Hostname, "h2")
		}
	})

	t.Run("empty JSON array returns empty nodes", func(t *testing.T) {
		nodes, err := parseNodesJSON([]byte(`[]`))
		if err != nil {
			t.Fatalf("parseNodesJSON() error = %v", err)
		}
		if len(nodes) != 0 {
			t.Errorf("expected 0 nodes, got %d", len(nodes))
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := parseNodesJSON([]byte(`not json`))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("missing optional IP field handled gracefully", func(t *testing.T) {
		input := `[{"kind":"node","metadata":{"labels":{}},"spec":{"hostname":"h1","cmd_labels":{"os":{"result":"Linux"}}}}]`
		nodes, err := parseNodesJSON([]byte(input))
		if err != nil {
			t.Fatalf("parseNodesJSON() error = %v", err)
		}
		if len(nodes) != 1 {
			t.Fatalf("expected 1 node, got %d", len(nodes))
		}
		if nodes[0].IP != "" {
			t.Errorf("expected empty IP, got %q", nodes[0].IP)
		}
	})
}

func Test_parseClusterJSON(t *testing.T) {
	t.Run("valid status returns cluster name", func(t *testing.T) {
		data, err := os.ReadFile("testdata/tsh-status.json")
		if err != nil {
			t.Fatalf("failed to read fixture: %v", err)
		}
		cluster, err := parseClusterJSON(data)
		if err != nil {
			t.Fatalf("parseClusterJSON() error = %v", err)
		}
		if cluster != "teleport.example.com" {
			t.Errorf("cluster = %q, want %q", cluster, "teleport.example.com")
		}
	})

	t.Run("missing active key returns error", func(t *testing.T) {
		input := `{"other":"data"}`
		_, err := parseClusterJSON([]byte(input))
		if err == nil {
			t.Error("expected error for missing active key")
		}
	})

	t.Run("missing cluster field returns error", func(t *testing.T) {
		input := `{"active":{"username":"user@example.com"}}`
		_, err := parseClusterJSON([]byte(input))
		if err == nil {
			t.Error("expected error for missing cluster field")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := parseClusterJSON([]byte(`not json`))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}
