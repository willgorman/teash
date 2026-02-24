package tui

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/willgorman/teash/internal/app"
	"github.com/willgorman/teash/internal/cache"
	"github.com/willgorman/teash/internal/labels"
	"github.com/willgorman/teash/internal/teleport"
)

func newTestModel(t *testing.T) Model {
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
			}, nil
		},
	}

	cacheStore := &cache.FileStore{BaseDir: t.TempDir()}
	labelStore := &labels.FileStore{BaseDir: t.TempDir()}

	svc := app.NewService(mock, cacheStore, labelStore)
	svc.SetCluster("test-cluster")

	// Pre-populate cache so LoadServers works
	rawNodes, _ := mock.ListNodes(context.Background())
	cacheStore.Put(&cache.Entry{
		Cluster:   "test-cluster",
		UpdatedAt: time.Now(),
		RawNodes:  rawNodes,
	})

	return New(svc)
}

func TestModel_ShowsServers(t *testing.T) {
	m := newTestModel(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Wait for actual server data to appear (hostnames start with "dc1a")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "dc1a")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_ShowsClusterInStatus(t *testing.T) {
	m := newTestModel(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "test-cluster")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_QuitOnQ(t *testing.T) {
	m := newTestModel(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Wait for actual server data to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "dc1a")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestModel_EmptyServerList(t *testing.T) {
	mock := &teleport.MockClient{
		ListNodesFunc: func(ctx context.Context) (json.RawMessage, error) {
			return json.RawMessage(`[]`), nil
		},
		GetStatusFunc: func(ctx context.Context) (*teleport.Status, error) {
			return &teleport.Status{
				Active: &teleport.Profile{
					Cluster:    "empty-cluster",
					ValidUntil: "2099-01-01T00:00:00Z",
				},
			}, nil
		},
	}

	cacheStore := &cache.FileStore{BaseDir: t.TempDir()}
	labelStore := &labels.FileStore{BaseDir: t.TempDir()}

	svc := app.NewService(mock, cacheStore, labelStore)
	svc.SetCluster("empty-cluster")

	// Pre-populate with empty cache
	cacheStore.Put(&cache.Entry{
		Cluster:   "empty-cluster",
		UpdatedAt: time.Now(),
		RawNodes:  json.RawMessage(`[]`),
	})

	m := New(svc)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "No servers found")
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m"},
		{5 * time.Minute, "5m"},
		{65 * time.Minute, "1h5m"},
		{2*time.Hour + 30*time.Minute, "2h30m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
