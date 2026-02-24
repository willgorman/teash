package teleport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestMockClient_ListNodes(t *testing.T) {
	fixture, err := os.ReadFile("../../example-data/cluster-a.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	mock := &MockClient{
		ListNodesFunc: func(ctx context.Context) (json.RawMessage, error) {
			return json.RawMessage(fixture), nil
		},
	}

	data, err := mock.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}

	servers, err := ParseNodesJSON(data)
	if err != nil {
		t.Fatalf("parsing nodes: %v", err)
	}

	if len(servers) == 0 {
		t.Error("expected servers from mock")
	}
}

func TestMockClient_GetStatus(t *testing.T) {
	mock := &MockClient{
		GetStatusFunc: func(ctx context.Context) (*Status, error) {
			return &Status{
				Active: &Profile{
					Cluster:    "test-cluster",
					Username:   "user@example.com",
					ValidUntil: "2099-01-01T00:00:00Z",
				},
			}, nil
		},
	}

	status, err := mock.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if !status.IsLoggedIn() {
		t.Error("expected logged in")
	}
	if status.ActiveCluster() != "test-cluster" {
		t.Errorf("expected 'test-cluster', got %q", status.ActiveCluster())
	}
}

func TestMockClient_ListNodesError(t *testing.T) {
	mock := &MockClient{
		ListNodesFunc: func(ctx context.Context) (json.RawMessage, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	_, err := mock.ListNodes(context.Background())
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockClient_Defaults(t *testing.T) {
	mock := &MockClient{}

	data, err := mock.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("default ListNodes: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("expected empty array, got %s", data)
	}

	status, err := mock.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("default GetStatus: %v", err)
	}
	if status.ActiveCluster() != "" {
		t.Errorf("expected empty cluster, got %q", status.ActiveCluster())
	}

	cmd := mock.LoginCmd("test")
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}

	cmd = mock.SSHCmd("host", "user")
	if cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

// Verify TshClient implements Client interface at compile time.
var _ Client = (*TshClient)(nil)
var _ Client = (*MockClient)(nil)
