package teleport

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Client defines the interface for interacting with Teleport via tsh.
type Client interface {
	// ListNodes returns the raw JSON output of `tsh ls --format=json`.
	ListNodes(ctx context.Context) (json.RawMessage, error)

	// GetStatus returns the parsed output of `tsh status --format=json`.
	GetStatus(ctx context.Context) (*Status, error)

	// LoginCmd returns an *exec.Cmd for `tsh login` to the given cluster.
	LoginCmd(cluster string) *exec.Cmd

	// SSHCmd returns an *exec.Cmd for `tsh ssh` to the given host with the given login.
	SSHCmd(host, login string) *exec.Cmd
}

// TshClient implements Client by shelling out to the tsh binary.
type TshClient struct {
	// TshPath is the path to the tsh binary. Defaults to "tsh".
	TshPath string
}

// NewTshClient creates a TshClient using the default tsh binary.
func NewTshClient() *TshClient {
	return &TshClient{TshPath: "tsh"}
}

func (c *TshClient) tsh() string {
	if c.TshPath != "" {
		return c.TshPath
	}
	return "tsh"
}

func (c *TshClient) ListNodes(ctx context.Context) (json.RawMessage, error) {
	cmd := exec.CommandContext(ctx, c.tsh(), "ls", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("tsh ls failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("tsh ls: %w", err)
	}
	return json.RawMessage(out), nil
}

func (c *TshClient) GetStatus(ctx context.Context) (*Status, error) {
	cmd := exec.CommandContext(ctx, c.tsh(), "status", "--format=json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("tsh status failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("tsh status: %w", err)
	}
	return ParseStatusJSON(out)
}

func (c *TshClient) LoginCmd(cluster string) *exec.Cmd {
	return exec.Command(c.tsh(), "login", cluster)
}

func (c *TshClient) SSHCmd(host, login string) *exec.Cmd {
	if login != "" {
		return exec.Command(c.tsh(), "ssh", fmt.Sprintf("%s@%s", login, host))
	}
	return exec.Command(c.tsh(), "ssh", host)
}
