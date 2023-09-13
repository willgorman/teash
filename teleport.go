package main

import (
	"encoding/json"
	"os/exec"

	"github.com/davecgh/go-spew/spew"
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
		spew.Dump(data)
	}
	return Nodes{
		{Hostname: "foo"},
		{Hostname: "bar"},
	}, nil
}

type Nodes []Node

type Node struct {
	Labels   map[string]string
	Hostname string
	IP       string
	OS       string
}

func lsNodesJson() (string, error) {
	cmd := exec.Command("tsh", "ls", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// TODO: (willgorman) handle different error types (not logged in, etc)
		return "", err
	}
	return string(output), nil
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
