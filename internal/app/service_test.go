package app

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/willgorman/teash/internal/cache"
	"github.com/willgorman/teash/internal/labels"
	"github.com/willgorman/teash/internal/teleport"
)

func newTestService(t *testing.T) (*Service, *teleport.MockClient) {
	t.Helper()
	fixture, err := os.ReadFile("../../example-data/cluster-a.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	mock := &teleport.MockClient{
		ListNodesFunc: func(ctx context.Context) (json.RawMessage, error) {
			return json.RawMessage(fixture), nil
		},
		GetStatusFunc: func(ctx context.Context) (*teleport.Status, error) {
			return &teleport.Status{
				Active: &teleport.Profile{
					Cluster:    "test-cluster",
					Username:   "user@example.com",
					ValidUntil: "2099-01-01T00:00:00Z",
					Logins:     []string{"testuser"},
				},
				Profiles: []teleport.Profile{
					{Cluster: "other-cluster"},
				},
			}, nil
		},
	}

	cacheStore := &cache.FileStore{BaseDir: t.TempDir()}
	labelStore := &labels.FileStore{BaseDir: t.TempDir()}

	svc := NewService(mock, cacheStore, labelStore)
	return svc, mock
}

func TestService_DetectCluster(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.DetectCluster(context.Background()); err != nil {
		t.Fatalf("DetectCluster: %v", err)
	}

	if svc.ActiveCluster() != "test-cluster" {
		t.Errorf("expected 'test-cluster', got %q", svc.ActiveCluster())
	}
	if !svc.IsLoggedIn() {
		t.Error("expected logged in")
	}
}

func TestService_AvailableClusters(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.DetectCluster(context.Background()); err != nil {
		t.Fatalf("DetectCluster: %v", err)
	}

	clusters := svc.AvailableClusters()
	if len(clusters) != 2 {
		t.Errorf("expected 2 clusters, got %d: %v", len(clusters), clusters)
	}
}

func TestService_RefreshAndLoadServers(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	servers, err := svc.RefreshServers(context.Background())
	if err != nil {
		t.Fatalf("RefreshServers: %v", err)
	}
	if len(servers) == 0 {
		t.Fatal("expected servers after refresh")
	}

	// Verify cache was populated
	loaded, err := svc.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers: %v", err)
	}
	if len(loaded) != len(servers) {
		t.Errorf("expected %d servers from cache, got %d", len(servers), len(loaded))
	}
}

func TestService_LoadServers_EmptyCache(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	servers, err := svc.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers: %v", err)
	}
	if servers != nil {
		t.Errorf("expected nil for empty cache, got %d servers", len(servers))
	}
}

func TestService_SetLabel_MergesWithServers(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	// Refresh to populate cache
	servers, err := svc.RefreshServers(context.Background())
	if err != nil {
		t.Fatalf("RefreshServers: %v", err)
	}

	hostname := servers[0].Hostname

	// Set a user label
	if err := svc.SetLabel(hostname, "custom-env", "testing"); err != nil {
		t.Fatalf("SetLabel: %v", err)
	}

	// Reload and verify
	loaded, err := svc.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers: %v", err)
	}

	var found bool
	for _, sv := range loaded {
		if sv.Hostname == hostname {
			found = true
			if sv.AllLabels["custom-env"] != "testing" {
				t.Errorf("expected custom-env=testing in AllLabels, got %q", sv.AllLabels["custom-env"])
			}
			if sv.UserLabels["custom-env"] != "testing" {
				t.Errorf("expected custom-env=testing in UserLabels, got %q", sv.UserLabels["custom-env"])
			}
			break
		}
	}
	if !found {
		t.Errorf("server %q not found in loaded servers", hostname)
	}
}

func TestService_AllLabelKeys(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	servers, err := svc.RefreshServers(context.Background())
	if err != nil {
		t.Fatalf("RefreshServers: %v", err)
	}

	keys := svc.AllLabelKeys(servers)
	if len(keys) == 0 {
		t.Error("expected label keys")
	}

	// Should include standard metadata labels
	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, expected := range []string{"env", "region"} {
		if !found[expected] {
			t.Errorf("expected key %q in AllLabelKeys", expected)
		}
	}

	// Should be sorted
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Errorf("keys not sorted: %q after %q", keys[i], keys[i-1])
		}
	}
}

func TestService_AllLabelKeys_IncludesUserLabels(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	servers, err := svc.RefreshServers(context.Background())
	if err != nil {
		t.Fatalf("RefreshServers: %v", err)
	}

	if err := svc.SetLabel(servers[0].Hostname, "custom-key", "value"); err != nil {
		t.Fatalf("SetLabel: %v", err)
	}

	// Reload to get user labels
	loaded, err := svc.LoadServers()
	if err != nil {
		t.Fatalf("LoadServers: %v", err)
	}

	keys := svc.AllLabelKeys(loaded)
	found := false
	for _, k := range keys {
		if k == "custom-key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'custom-key' in AllLabelKeys after SetLabel")
	}
}

func TestService_ImportLabels(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	data := labels.ImportFile{
		"test-cluster": {
			"server1": {"env": "prod"},
			"server2": {"role": "web"},
		},
	}

	if err := svc.ImportLabels(data); err != nil {
		t.Fatalf("ImportLabels: %v", err)
	}

	exported, err := svc.ExportLabels()
	if err != nil {
		t.Fatalf("ExportLabels: %v", err)
	}

	cl := exported["test-cluster"]
	if cl["server1"]["env"] != "prod" {
		t.Error("server1 env mismatch")
	}
	if cl["server2"]["role"] != "web" {
		t.Error("server2 role mismatch")
	}
}

func TestService_ImportLabels_SingleClusterFallback(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	// Import with a different cluster name, but only one key
	data := labels.ImportFile{
		"other-name": {
			"server1": {"env": "staging"},
		},
	}

	if err := svc.ImportLabels(data); err != nil {
		t.Fatalf("ImportLabels with single cluster: %v", err)
	}

	exported, err := svc.ExportLabels()
	if err != nil {
		t.Fatalf("ExportLabels: %v", err)
	}

	if exported["test-cluster"]["server1"]["env"] != "staging" {
		t.Error("expected fallback to single cluster data")
	}
}

func TestService_CacheLastUpdated(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetCluster("test-cluster")

	// Before refresh: zero time
	updated, err := svc.CacheLastUpdated()
	if err != nil {
		t.Fatalf("CacheLastUpdated: %v", err)
	}
	if !updated.IsZero() {
		t.Errorf("expected zero time before refresh, got %v", updated)
	}

	// After refresh: non-zero
	if _, err := svc.RefreshServers(context.Background()); err != nil {
		t.Fatalf("RefreshServers: %v", err)
	}

	updated, err = svc.CacheLastUpdated()
	if err != nil {
		t.Fatalf("CacheLastUpdated: %v", err)
	}
	if updated.IsZero() {
		t.Error("expected non-zero time after refresh")
	}
}

func TestService_DefaultLogin(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.DetectCluster(context.Background()); err != nil {
		t.Fatalf("DetectCluster: %v", err)
	}

	if login := svc.DefaultLogin(); login != "testuser" {
		t.Errorf("expected 'testuser', got %q", login)
	}
}

func TestService_SetCluster(t *testing.T) {
	svc, _ := newTestService(t)

	svc.SetCluster("override-cluster")
	if svc.ActiveCluster() != "override-cluster" {
		t.Errorf("expected 'override-cluster', got %q", svc.ActiveCluster())
	}
}
