package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"time"

	"github.com/willgorman/teash/internal/cache"
	"github.com/willgorman/teash/internal/labels"
	"github.com/willgorman/teash/internal/teleport"
)

// ServerView is an enriched server representation for display.
type ServerView struct {
	teleport.Server
	UserLabels map[string]string // user-defined labels only
	AllLabels  map[string]string // merged: server labels + user labels
}

// Service orchestrates teleport client, cache, and label operations.
type Service struct {
	client  teleport.Client
	cache   cache.Store
	labels  labels.Store
	status  *teleport.Status
	cluster string
}

// NewService creates a new Service with the given dependencies.
func NewService(client teleport.Client, cacheStore cache.Store, labelStore labels.Store) *Service {
	return &Service{
		client: client,
		cache:  cacheStore,
		labels: labelStore,
	}
}

// DetectCluster calls tsh status to determine the active cluster.
func (s *Service) DetectCluster(ctx context.Context) error {
	status, err := s.client.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("detecting cluster: %w", err)
	}
	s.status = status
	s.cluster = status.ActiveCluster()
	return nil
}

// SetCluster overrides the active cluster (e.g. from a CLI flag).
func (s *Service) SetCluster(cluster string) {
	s.cluster = cluster
}

// IsLoggedIn returns whether the current session is valid.
func (s *Service) IsLoggedIn() bool {
	return s.status != nil && s.status.IsLoggedIn()
}

// ActiveCluster returns the current cluster name.
func (s *Service) ActiveCluster() string {
	return s.cluster
}

// AvailableClusters returns all known cluster names.
func (s *Service) AvailableClusters() []string {
	if s.status == nil {
		return nil
	}
	return s.status.AvailableClusters()
}

// DefaultLogin returns the first available login for the active profile.
func (s *Service) DefaultLogin() string {
	if s.status != nil && s.status.Active != nil && len(s.status.Active.Logins) > 0 {
		return s.status.Active.Logins[0]
	}
	return ""
}

// LoadServers loads servers from the cache and merges user labels.
func (s *Service) LoadServers() ([]ServerView, error) {
	entry, err := s.cache.Get(s.cluster)
	if err != nil {
		return nil, fmt.Errorf("loading cache: %w", err)
	}
	if entry == nil {
		return nil, nil
	}

	servers, err := teleport.ParseNodesJSON(entry.RawNodes)
	if err != nil {
		return nil, fmt.Errorf("parsing cached nodes: %w", err)
	}

	return s.enrichServers(servers)
}

// RefreshServers fetches fresh server data from tsh, updates the cache, and returns enriched views.
func (s *Service) RefreshServers(ctx context.Context) ([]ServerView, error) {
	rawNodes, err := s.client.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	entry := &cache.Entry{
		Cluster:   s.cluster,
		UpdatedAt: time.Now(),
		RawNodes:  rawNodes,
	}
	if err := s.cache.Put(entry); err != nil {
		return nil, fmt.Errorf("updating cache: %w", err)
	}

	servers, err := teleport.ParseNodesJSON(rawNodes)
	if err != nil {
		return nil, fmt.Errorf("parsing nodes: %w", err)
	}

	return s.enrichServers(servers)
}

func (s *Service) enrichServers(servers []teleport.Server) ([]ServerView, error) {
	userLabels, err := s.labels.GetAll(s.cluster)
	if err != nil {
		return nil, fmt.Errorf("loading user labels: %w", err)
	}

	views := make([]ServerView, len(servers))
	for i, srv := range servers {
		ul := userLabels[srv.Hostname]
		allLabels := make(map[string]string, len(srv.Labels)+len(ul))
		for k, v := range srv.Labels {
			allLabels[k] = v
		}
		for k, v := range ul {
			allLabels[k] = v
		}
		views[i] = ServerView{
			Server:     srv,
			UserLabels: ul,
			AllLabels:  allLabels,
		}
	}
	return views, nil
}

// CacheLastUpdated returns when the cache was last updated for the current cluster.
func (s *Service) CacheLastUpdated() (time.Time, error) {
	return s.cache.LastUpdated(s.cluster)
}

// SetLabel sets a user-defined label on a server.
func (s *Service) SetLabel(hostname, key, value string) error {
	return s.labels.Set(s.cluster, hostname, key, value)
}

// DeleteLabel removes a user-defined label from a server.
func (s *Service) DeleteLabel(hostname, key string) error {
	return s.labels.Delete(s.cluster, hostname, key)
}

// ImportLabels merges imported label data for the current cluster.
func (s *Service) ImportLabels(data labels.ImportFile) error {
	clusterLabels, ok := data[s.cluster]
	if !ok {
		// If there's only one key, use it regardless of cluster name
		if len(data) == 1 {
			for _, v := range data {
				clusterLabels = v
			}
		} else {
			return fmt.Errorf("no labels found for cluster %q", s.cluster)
		}
	}
	return s.labels.MergeAll(s.cluster, clusterLabels)
}

// ExportLabels returns all user labels for the current cluster in import file format.
func (s *Service) ExportLabels() (labels.ImportFile, error) {
	sl, err := s.labels.GetAll(s.cluster)
	if err != nil {
		return nil, err
	}
	return labels.ImportFile{s.cluster: sl}, nil
}

// AllLabelKeys returns sorted unique label keys from all servers and user labels.
func (s *Service) AllLabelKeys(servers []ServerView) []string {
	keys := make(map[string]bool)
	for _, sv := range servers {
		for k := range sv.AllLabels {
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

// LoginCmd returns an exec.Cmd for tsh login to the current cluster.
func (s *Service) LoginCmd() *exec.Cmd {
	return s.client.LoginCmd(s.cluster)
}

// SSHCmd returns an exec.Cmd for tsh ssh to the given host.
func (s *Service) SSHCmd(hostname, login string) *exec.Cmd {
	return s.client.SSHCmd(hostname, login)
}

// RawCacheData returns the raw JSON from the cache for the current cluster.
func (s *Service) RawCacheData() (json.RawMessage, error) {
	entry, err := s.cache.Get(s.cluster)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	return entry.RawNodes, nil
}
