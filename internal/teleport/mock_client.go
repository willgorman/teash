package teleport

import (
	"context"
	"encoding/json"
	"os/exec"
)

// MockClient implements Client for testing purposes.
type MockClient struct {
	ListNodesFunc func(ctx context.Context) (json.RawMessage, error)
	GetStatusFunc func(ctx context.Context) (*Status, error)
	LoginCmdFunc  func(cluster string) *exec.Cmd
	SSHCmdFunc    func(host, login string) *exec.Cmd
}

func (m *MockClient) ListNodes(ctx context.Context) (json.RawMessage, error) {
	if m.ListNodesFunc != nil {
		return m.ListNodesFunc(ctx)
	}
	return json.RawMessage("[]"), nil
}

func (m *MockClient) GetStatus(ctx context.Context) (*Status, error) {
	if m.GetStatusFunc != nil {
		return m.GetStatusFunc(ctx)
	}
	return &Status{}, nil
}

func (m *MockClient) LoginCmd(cluster string) *exec.Cmd {
	if m.LoginCmdFunc != nil {
		return m.LoginCmdFunc(cluster)
	}
	return exec.Command("echo", "mock-login", cluster)
}

func (m *MockClient) SSHCmd(host, login string) *exec.Cmd {
	if m.SSHCmdFunc != nil {
		return m.SSHCmdFunc(host, login)
	}
	return exec.Command("echo", "mock-ssh", host)
}
