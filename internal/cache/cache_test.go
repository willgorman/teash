package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	return &FileStore{BaseDir: t.TempDir()}
}

func TestFileStore_Roundtrip(t *testing.T) {
	store := newTestStore(t)
	now := time.Now().Truncate(time.Millisecond)

	entry := &Entry{
		Cluster:   "test-cluster",
		UpdatedAt: now,
		RawNodes:  json.RawMessage(`[{"kind":"node"}]`),
	}

	if err := store.Put(entry); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := store.Get("test-cluster")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.Cluster != "test-cluster" {
		t.Errorf("expected cluster 'test-cluster', got %q", got.Cluster)
	}
	if !got.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, got.UpdatedAt)
	}
	if string(got.RawNodes) != `[{"kind":"node"}]` {
		t.Errorf("unexpected RawNodes: %s", got.RawNodes)
	}
}

func TestFileStore_GetMissing(t *testing.T) {
	store := newTestStore(t)

	entry, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil for missing cluster, got %+v", entry)
	}
}

func TestFileStore_Delete(t *testing.T) {
	store := newTestStore(t)

	entry := &Entry{
		Cluster:   "to-delete",
		UpdatedAt: time.Now(),
		RawNodes:  json.RawMessage(`[]`),
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := store.Get("to-delete")
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestFileStore_DeleteMissing(t *testing.T) {
	store := newTestStore(t)

	if err := store.Delete("nonexistent"); err != nil {
		t.Errorf("deleting nonexistent should not error: %v", err)
	}
}

func TestFileStore_MultipleClusters(t *testing.T) {
	store := newTestStore(t)

	for _, cluster := range []string{"cluster-a", "cluster-b", "cluster-c"} {
		entry := &Entry{
			Cluster:   cluster,
			UpdatedAt: time.Now(),
			RawNodes:  json.RawMessage(`[{"cluster":"` + cluster + `"}]`),
		}
		if err := store.Put(entry); err != nil {
			t.Fatalf("Put %s: %v", cluster, err)
		}
	}

	for _, cluster := range []string{"cluster-a", "cluster-b", "cluster-c"} {
		got, err := store.Get(cluster)
		if err != nil {
			t.Fatalf("Get %s: %v", cluster, err)
		}
		if got == nil {
			t.Fatalf("expected entry for %s", cluster)
		}
		if got.Cluster != cluster {
			t.Errorf("expected cluster %q, got %q", cluster, got.Cluster)
		}
	}
}

func TestFileStore_LastUpdated(t *testing.T) {
	store := newTestStore(t)

	// Missing cluster returns zero time
	updated, err := store.LastUpdated("missing")
	if err != nil {
		t.Fatalf("LastUpdated: %v", err)
	}
	if !updated.IsZero() {
		t.Errorf("expected zero time for missing cluster, got %v", updated)
	}

	// After put, returns the timestamp
	now := time.Now().Truncate(time.Millisecond)
	entry := &Entry{
		Cluster:   "timed",
		UpdatedAt: now,
		RawNodes:  json.RawMessage(`[]`),
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("Put: %v", err)
	}

	updated, err = store.LastUpdated("timed")
	if err != nil {
		t.Fatalf("LastUpdated: %v", err)
	}
	if !updated.Equal(now) {
		t.Errorf("expected %v, got %v", now, updated)
	}
}

func TestFileStore_OverwriteExisting(t *testing.T) {
	store := newTestStore(t)

	entry1 := &Entry{
		Cluster:   "overwrite",
		UpdatedAt: time.Now(),
		RawNodes:  json.RawMessage(`[{"v":1}]`),
	}
	if err := store.Put(entry1); err != nil {
		t.Fatalf("Put 1: %v", err)
	}

	entry2 := &Entry{
		Cluster:   "overwrite",
		UpdatedAt: time.Now().Add(time.Hour),
		RawNodes:  json.RawMessage(`[{"v":2}]`),
	}
	if err := store.Put(entry2); err != nil {
		t.Fatalf("Put 2: %v", err)
	}

	got, err := store.Get("overwrite")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.RawNodes) != `[{"v":2}]` {
		t.Errorf("expected updated data, got %s", got.RawNodes)
	}
}

func TestFileStore_CorruptedCache(t *testing.T) {
	store := newTestStore(t)

	// Write garbage to the cache file
	if err := os.MkdirAll(store.BaseDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(store.BaseDir, "corrupt.json")
	if err := os.WriteFile(path, []byte("not json at all {{{"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Should return nil (treat as empty), not an error
	entry, err := store.Get("corrupt")
	if err != nil {
		t.Errorf("expected nil error for corrupt cache, got: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil entry for corrupt cache, got: %+v", entry)
	}
}

func TestFileStore_ClusterNameSanitization(t *testing.T) {
	store := newTestStore(t)

	entry := &Entry{
		Cluster:   "cluster/with:special chars",
		UpdatedAt: time.Now(),
		RawNodes:  json.RawMessage(`[]`),
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := store.Get("cluster/with:special chars")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry")
	}
}
