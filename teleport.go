package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type tshWrapper struct {
	nodes   Nodes
	tshPath string
}

func NewTeleport() (Teleport, error) {
	if demoMode := os.Getenv("TEASH_DEMO"); demoMode != "" {
		sshPath, _ := exec.LookPath("ssh")
		if sshPath == "" {
			return nil, errors.New("`ssh` command not found")
		}
		return &demo{sshPath: sshPath, demoServer: demoMode}, nil
	}
	tsh, _ = exec.LookPath("tsh")
	if tsh == "" {
		return nil, errors.New("teleport `tsh` command not found")
	}
	return &tshWrapper{
		nodes:   Nodes{},
		tshPath: tsh,
	}, nil
}

func (t *tshWrapper) Connect(cmd []string) {
	err := syscall.Exec(t.tshPath, cmd, os.Environ())
	if err != nil {
		panic(err)
	}
}

func (t *tshWrapper) GetNodes(refresh bool) (Nodes, error) {
	if len(t.nodes) == 0 || refresh {
		data := []struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Spec struct {
				Hostname  string `json:"hostname"`
				CmdLabels struct {
					Ip struct {
						Result string `json:"result"`
					} `json:"ip"`
					Os struct {
						Result string `json:"result"`
					} `json:"os"`
				} `json:"cmd_labels"`
			} `json:"spec"`
		}{}
		jsonNodes, err := lsNodesJson()
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(jsonNodes), &data)
		if err != nil {
			return nil, err
		}
		t.nodes = Nodes{}
		for _, n := range data {
			if n.Kind != "node" {
				continue
			}
			t.nodes = append(t.nodes, Node{
				Labels:   n.Metadata.Labels,
				Hostname: n.Spec.Hostname,
				IP:       n.Spec.CmdLabels.Ip.Result,
				OS:       n.Spec.CmdLabels.Os.Result,
			})
		}
	}
	return t.nodes, nil
}

type Nodes []Node

type Node struct {
	Labels   map[string]string
	Hostname string
	IP       string
	OS       string
}

func (t *tshWrapper) GetCluster() (string, error) {
	cmd := exec.Command("tsh", "status", "--format=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "Not logged in") {
			return "", fmt.Errorf("%s Run `tsh login` first", strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("%s: %s", err, string(output))
	}

	status := map[string]any{}
	if err := json.Unmarshal(output, &status); err != nil {
		return "", fmt.Errorf("`tsh status` returned invalid data, cannot check login:\n%s", string(output))
	}

	// i _think_ that even if the active profile is expired it's still going to be here
	active, ok := status["active"].(map[string]any)
	if !ok {
		return "", errors.New("no active profile found, `tsh login` and try again")
	}

	cluster, ok := active["cluster"].(string)
	if !ok {
		return "", errors.New("no active cluster found, `tsh login` and try again")
	}

	return cluster, nil
}

func lsNodesJson() (string, error) {
	cmd := exec.Command("tsh", "ls", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	// if `tsh ls` has to re-login first then it returns an extra bit of
	// text in front of the json so we need to remove that
	return string(stripInvalidJSONPrefix(output)), nil
}

type teleportItem struct {
	Kind     string
	Metadata metadata
}

type metadata struct {
	Name   string
	Labels map[string]string
}

type spec struct {
	Hostname string
}

type cmdLabels struct{}

type Result struct {
	Result string
}

// given data that should contain valid json but prefixed with
// arbitrary text, return the string without the invalid json
// prefix
func stripInvalidJSONPrefix(data []byte) []byte {
	for {
		if json.Valid(data) {
			return data
		}
		if len(data) == 0 {
			return data
		}
		data = data[1:]
	}
}

type demo struct {
	demoServer string
	sshPath    string
}

func (d *demo) Connect(_ []string) {
	log.Println(d.sshPath, d.demoServer)
	err := syscall.Exec(d.sshPath, []string{"ssh", d.demoServer}, os.Environ())
	if err != nil {
		panic(err)
	}
}

func (d *demo) GetCluster() (string, error) {
	return "demo-cluster", nil
}

func (d *demo) GetNodes(refresh bool) (Nodes, error) {
	return Nodes{
		Node{
			Hostname: "host1.example.com",
			IP:       "192.168.1.1",
			OS:       "Ubuntu 22.04",
			Labels: map[string]string{
				"Team": "dev",
				"AZ":   "us-east-1a",
			},
		},
		Node{
			Hostname: "host2.example.com",
			IP:       "192.168.1.2",
			OS:       "Ubuntu 22.04",
			Labels: map[string]string{
				"Team": "dev",
				"AZ":   "us-east-1b",
			},
		},
		Node{
			Hostname: "host3.example.com",
			IP:       "192.168.1.3",
			OS:       "Ubuntu 22.04",
			Labels: map[string]string{
				"Team": "dev",
				"AZ":   "us-east-1b",
			},
		},
		Node{
			Hostname: "host4.example.com",
			IP:       "192.168.1.4",
			OS:       "CentOS Stream",
			Labels: map[string]string{
				"Team": "infra",
				"AZ":   "us-east-1b",
			},
		},
		Node{
			Hostname: "host5.example.com",
			IP:       "192.168.1.5",
			OS:       "CentOS Stream",
			Labels: map[string]string{
				"Team": "infra",
				"AZ":   "us-east-1a",
			},
		},
		Node{
			Hostname: "host6.example.com",
			IP:       "192.168.1.6",
			OS:       "NixOS 23.11",
			Labels: map[string]string{
				"Team": "infra",
				"AZ":   "us-east-1c",
			},
		},
		Node{
			Hostname: "host7.example.com",
			IP:       "192.168.1.7",
			OS:       "NixOS 23.11",
			Labels: map[string]string{
				"Team": "infra",
				"AZ":   "us-east-1c",
			},
		},
		Node{
			Hostname: "host8.example.com",
			IP:       "192.168.1.8",
			OS:       "Rocky Linux 9",
			Labels: map[string]string{
				"Team": "dev",
				"AZ":   "us-east-1a",
			},
		},
	}, nil
}
