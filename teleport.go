package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type Teleport struct {
	nodes Nodes
}

func New() *Teleport {
	return &Teleport{
		nodes: Nodes{},
	}
}

func (t *Teleport) GetNodes(refresh bool) (Nodes, error) {
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

func CheckProfiles() error {
	cmd := exec.Command("tsh", "status", "--format=json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "Not logged in") {
			return fmt.Errorf("%s Run `tsh login` first", strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("%s: %s", err, string(output))
	}

	status := map[string]any{}
	if err := json.Unmarshal(output, &status); err != nil {
		return fmt.Errorf("`tsh status` returned invalid data, cannot check login:\n%s", string(output))
	}

	// i _think_ that even if the active profile is expired it's still going to be here
	if _, ok := status["active"]; !ok {
		return errors.New("no active profile found, `tsh login` and try again")
	}
	return nil
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
