package labels

import (
	"testing"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	return &FileStore{BaseDir: t.TempDir()}
}

func TestFileStore_SetAndGet(t *testing.T) {
	store := newTestStore(t)

	if err := store.Set("cluster-a", "server1", "env", "production"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	labels, err := store.GetForServer("cluster-a", "server1")
	if err != nil {
		t.Fatalf("GetForServer: %v", err)
	}
	if labels["env"] != "production" {
		t.Errorf("expected 'production', got %q", labels["env"])
	}
}

func TestFileStore_GetAll(t *testing.T) {
	store := newTestStore(t)

	if err := store.Set("cluster-a", "server1", "env", "production"); err != nil {
		t.Fatalf("Set 1: %v", err)
	}
	if err := store.Set("cluster-a", "server2", "role", "web"); err != nil {
		t.Fatalf("Set 2: %v", err)
	}

	all, err := store.GetAll("cluster-a")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 servers, got %d", len(all))
	}
	if all["server1"]["env"] != "production" {
		t.Error("server1 env mismatch")
	}
	if all["server2"]["role"] != "web" {
		t.Error("server2 role mismatch")
	}
}

func TestFileStore_Delete(t *testing.T) {
	store := newTestStore(t)

	if err := store.Set("cluster-a", "server1", "env", "production"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Set("cluster-a", "server1", "role", "web"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := store.Delete("cluster-a", "server1", "env"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	labels, err := store.GetForServer("cluster-a", "server1")
	if err != nil {
		t.Fatalf("GetForServer: %v", err)
	}
	if _, ok := labels["env"]; ok {
		t.Error("expected 'env' to be deleted")
	}
	if labels["role"] != "web" {
		t.Error("expected 'role' to remain")
	}
}

func TestFileStore_DeleteLastKey_RemovesServer(t *testing.T) {
	store := newTestStore(t)

	if err := store.Set("cluster-a", "server1", "env", "production"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Delete("cluster-a", "server1", "env"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	all, err := store.GetAll("cluster-a")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if _, ok := all["server1"]; ok {
		t.Error("expected server1 to be removed when all labels deleted")
	}
}

func TestFileStore_MultiClusterIsolation(t *testing.T) {
	store := newTestStore(t)

	if err := store.Set("cluster-a", "server1", "env", "prod"); err != nil {
		t.Fatalf("Set a: %v", err)
	}
	if err := store.Set("cluster-b", "server1", "env", "staging"); err != nil {
		t.Fatalf("Set b: %v", err)
	}

	labelsA, err := store.GetForServer("cluster-a", "server1")
	if err != nil {
		t.Fatalf("GetForServer a: %v", err)
	}
	labelsB, err := store.GetForServer("cluster-b", "server1")
	if err != nil {
		t.Fatalf("GetForServer b: %v", err)
	}

	if labelsA["env"] != "prod" {
		t.Errorf("cluster-a: expected 'prod', got %q", labelsA["env"])
	}
	if labelsB["env"] != "staging" {
		t.Errorf("cluster-b: expected 'staging', got %q", labelsB["env"])
	}
}

func TestFileStore_MergeAll(t *testing.T) {
	store := newTestStore(t)

	// Set initial data
	if err := store.Set("cluster-a", "server1", "env", "production"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Merge new data
	incoming := ServerLabels{
		"server1": {"role": "web"},           // add new key to existing server
		"server2": {"env": "staging"},        // add new server
	}
	if err := store.MergeAll("cluster-a", incoming); err != nil {
		t.Fatalf("MergeAll: %v", err)
	}

	all, err := store.GetAll("cluster-a")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	// server1 should have both keys
	if all["server1"]["env"] != "production" {
		t.Error("server1 env should be preserved")
	}
	if all["server1"]["role"] != "web" {
		t.Error("server1 role should be merged")
	}
	// server2 should exist
	if all["server2"]["env"] != "staging" {
		t.Error("server2 env should be present")
	}
}

func TestFileStore_ReplaceAll(t *testing.T) {
	store := newTestStore(t)

	if err := store.Set("cluster-a", "server1", "env", "production"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	replacement := ServerLabels{
		"server99": {"role": "db"},
	}
	if err := store.ReplaceAll("cluster-a", replacement); err != nil {
		t.Fatalf("ReplaceAll: %v", err)
	}

	all, err := store.GetAll("cluster-a")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if _, ok := all["server1"]; ok {
		t.Error("server1 should be replaced")
	}
	if all["server99"]["role"] != "db" {
		t.Error("server99 should exist with role=db")
	}
}

func TestFileStore_GetForServer_Missing(t *testing.T) {
	store := newTestStore(t)

	labels, err := store.GetForServer("cluster-a", "nonexistent")
	if err != nil {
		t.Fatalf("GetForServer: %v", err)
	}
	if labels != nil {
		t.Errorf("expected nil for missing server, got %v", labels)
	}
}

func TestFileStore_GetAll_EmptyCluster(t *testing.T) {
	store := newTestStore(t)

	all, err := store.GetAll("empty-cluster")
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected empty map, got %d entries", len(all))
	}
}
