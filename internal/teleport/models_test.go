package teleport

import (
	"os"
	"testing"
)

func TestParseNodesJSON(t *testing.T) {
	data, err := os.ReadFile("../../example-data/cluster-a.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	servers, err := ParseNodesJSON(data)
	if err != nil {
		t.Fatalf("parsing nodes: %v", err)
	}

	if len(servers) == 0 {
		t.Fatal("expected at least one server, got none")
	}

	// Verify first server fields
	first := servers[0]
	if first.ID == "" {
		t.Error("expected non-empty ID")
	}
	if first.Hostname == "" {
		t.Error("expected non-empty Hostname")
	}

	// Verify labels are populated
	if len(first.Labels) == 0 {
		t.Error("expected non-empty Labels")
	}

	// Check that metadata labels are included
	if _, ok := first.Labels["env"]; !ok {
		t.Error("expected 'env' label from metadata.labels")
	}
	if _, ok := first.Labels["region"]; !ok {
		t.Error("expected 'region' label from metadata.labels")
	}

	// Check that cmd_label results are extracted
	// "hostname" and "vm" cmd_labels should appear in Labels
	if _, ok := first.Labels["hostname"]; !ok {
		t.Error("expected 'hostname' from cmd_labels in Labels")
	}

	// OS should be populated from cmd_labels.os.result
	if first.OS == "" {
		t.Log("warning: OS is empty for first server (may be expected for some nodes)")
	}
}

func TestParseNodesJSON_AllServersHaveHostname(t *testing.T) {
	data, err := os.ReadFile("../../example-data/cluster-a.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	servers, err := ParseNodesJSON(data)
	if err != nil {
		t.Fatalf("parsing nodes: %v", err)
	}

	for i, s := range servers {
		if s.Hostname == "" {
			t.Errorf("server %d has empty hostname", i)
		}
		if s.ID == "" {
			t.Errorf("server %d has empty ID", i)
		}
	}
}

func TestAllLabelKeys(t *testing.T) {
	data, err := os.ReadFile("../../example-data/cluster-a.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	servers, err := ParseNodesJSON(data)
	if err != nil {
		t.Fatalf("parsing nodes: %v", err)
	}

	keys := AllLabelKeys(servers)
	if len(keys) == 0 {
		t.Fatal("expected label keys, got none")
	}

	// Should include metadata labels
	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, expected := range []string{"env", "region", "category1", "hostname", "teleport", "vm"} {
		if !found[expected] {
			t.Errorf("expected label key %q in AllLabelKeys", expected)
		}
	}

	// Should be sorted
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Errorf("keys not sorted: %q comes after %q", keys[i], keys[i-1])
		}
	}
}

func TestParseStatusJSON(t *testing.T) {
	data, err := os.ReadFile("../../example-data/status.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	status, err := ParseStatusJSON(data)
	if err != nil {
		t.Fatalf("parsing status: %v", err)
	}

	if status.Active == nil {
		t.Fatal("expected active profile")
	}
	if status.Active.Cluster != "dev-eu-cluster" {
		t.Errorf("expected cluster 'dev-eu-cluster', got %q", status.Active.Cluster)
	}
	if status.Active.Username != "jane.doe@acme.com" {
		t.Errorf("expected username 'jane.doe@acme.com', got %q", status.Active.Username)
	}
	if len(status.Active.Logins) == 0 {
		t.Error("expected at least one login")
	}
}

func TestStatus_ActiveCluster(t *testing.T) {
	status := &Status{
		Active: &Profile{Cluster: "test-cluster"},
	}
	if got := status.ActiveCluster(); got != "test-cluster" {
		t.Errorf("expected 'test-cluster', got %q", got)
	}

	empty := &Status{}
	if got := empty.ActiveCluster(); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestStatus_AvailableClusters(t *testing.T) {
	data, err := os.ReadFile("../../example-data/status.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	status, err := ParseStatusJSON(data)
	if err != nil {
		t.Fatalf("parsing status: %v", err)
	}

	clusters := status.AvailableClusters()
	if len(clusters) != 3 { // 1 active + 2 profiles
		t.Errorf("expected 3 clusters, got %d: %v", len(clusters), clusters)
	}

	// Active cluster should be first
	if clusters[0] != "dev-eu-cluster" {
		t.Errorf("expected active cluster first, got %q", clusters[0])
	}
}

func TestStatus_IsLoggedIn(t *testing.T) {
	// Expired session
	expired := &Status{
		Active: &Profile{
			Cluster:    "test",
			ValidUntil: "2020-01-01T00:00:00Z",
		},
	}
	if expired.IsLoggedIn() {
		t.Error("expected expired session to not be logged in")
	}

	// No active profile
	none := &Status{}
	if none.IsLoggedIn() {
		t.Error("expected no active profile to not be logged in")
	}

	// Valid session (far future)
	valid := &Status{
		Active: &Profile{
			Cluster:    "test",
			ValidUntil: "2099-01-01T00:00:00Z",
		},
	}
	if !valid.IsLoggedIn() {
		t.Error("expected valid session to be logged in")
	}
}

func TestParseNodes_EmptyInput(t *testing.T) {
	servers := ParseNodes(nil)
	if len(servers) != 0 {
		t.Errorf("expected empty result, got %d servers", len(servers))
	}
}

func TestParseNodesJSON_InvalidJSON(t *testing.T) {
	_, err := ParseNodesJSON([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
