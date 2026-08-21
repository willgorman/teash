package teleport

import (
	"encoding/json"
	"sort"
	"time"
)

// Node represents a single server node as returned by `tsh ls --format=json`.
type Node struct {
	Kind     string   `json:"kind"`
	Version  string   `json:"version"`
	Metadata Metadata `json:"metadata"`
	Spec     NodeSpec `json:"spec"`
}

// Metadata holds node metadata including labels.
type Metadata struct {
	Name     string            `json:"name"`
	Labels   map[string]string `json:"labels"`
	Expires  string            `json:"expires"`
	Revision string            `json:"revision"`
}

// NodeSpec holds the spec portion of a node.
type NodeSpec struct {
	Addr      string              `json:"addr"`
	Hostname  string              `json:"hostname"`
	CmdLabels map[string]CmdLabel `json:"cmd_labels"`
	UseTunnel bool                `json:"use_tunnel"`
	Version   string              `json:"version"`
}

// CmdLabel represents a dynamic command-based label.
type CmdLabel struct {
	Period  string   `json:"period"`
	Command []string `json:"command"`
	Result  string   `json:"result"`
}

// Status represents the output of `tsh status --format=json`.
type Status struct {
	Active   *Profile  `json:"active"`
	Profiles []Profile `json:"profiles"`
}

// Profile represents a Teleport profile/cluster.
type Profile struct {
	ProfileURL        string   `json:"profile_url"`
	Username          string   `json:"username"`
	Cluster           string   `json:"cluster"`
	Roles             []string `json:"roles"`
	Logins            []string `json:"logins"`
	KubernetesEnabled bool     `json:"kubernetes_enabled"`
	ValidUntil        string   `json:"valid_until"`
}

// IsLoggedIn returns true if there is an active profile with a valid session.
func (s *Status) IsLoggedIn() bool {
	if s.Active == nil || s.Active.Cluster == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, s.Active.ValidUntil)
	if err != nil {
		return false
	}
	return time.Now().Before(t)
}

// ActiveCluster returns the name of the currently active cluster, or empty string.
func (s *Status) ActiveCluster() string {
	if s.Active == nil {
		return ""
	}
	return s.Active.Cluster
}

// AvailableClusters returns the names of all known clusters (active + profiles).
func (s *Status) AvailableClusters() []string {
	seen := make(map[string]bool)
	var clusters []string
	if s.Active != nil && s.Active.Cluster != "" {
		seen[s.Active.Cluster] = true
		clusters = append(clusters, s.Active.Cluster)
	}
	for _, p := range s.Profiles {
		if p.Cluster != "" && !seen[p.Cluster] {
			seen[p.Cluster] = true
			clusters = append(clusters, p.Cluster)
		}
	}
	return clusters
}

// Server is a flattened, app-level representation of a Teleport server node.
type Server struct {
	ID       string            // metadata.name (UUID)
	Hostname string            // spec.hostname or cmd_labels.hostname.result
	Addr     string            // spec.addr or cmd_labels.ip.result
	OS       string            // cmd_labels.os.result
	Labels   map[string]string // merged: metadata.labels + cmd_label results (excluding hostname/ip/os)
}

// ParseNodes converts raw tsh Node objects into flattened Server structs.
func ParseNodes(nodes []Node) []Server {
	servers := make([]Server, 0, len(nodes))
	for _, n := range nodes {
		s := Server{
			ID:       n.Metadata.Name,
			Hostname: n.Spec.Hostname,
			Addr:     n.Spec.Addr,
			Labels:   make(map[string]string),
		}

		// Copy static metadata labels
		for k, v := range n.Metadata.Labels {
			s.Labels[k] = v
		}

		// Extract cmd_label results
		for k, cl := range n.Spec.CmdLabels {
			switch k {
			case "ip":
				if s.Addr == "" {
					s.Addr = cl.Result
				}
			case "os":
				s.OS = cl.Result
			case "hostname":
				if s.Hostname == "" {
					s.Hostname = cl.Result
				}
			default:
				s.Labels[k] = cl.Result
			}
		}

		servers = append(servers, s)
	}
	return servers
}

// ParseNodesJSON deserializes a JSON byte slice into Nodes, then converts to Servers.
func ParseNodesJSON(data []byte) ([]Server, error) {
	var nodes []Node
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, err
	}
	return ParseNodes(nodes), nil
}

// ParseStatusJSON deserializes a JSON byte slice into a Status struct.
func ParseStatusJSON(data []byte) (*Status, error) {
	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// AllLabelKeys returns a sorted list of all unique label keys across the given servers.
func AllLabelKeys(servers []Server) []string {
	keys := make(map[string]bool)
	for _, s := range servers {
		for k := range s.Labels {
			keys[k] = true
		}
	}
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
