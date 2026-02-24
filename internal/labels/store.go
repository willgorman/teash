package labels

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServerLabels maps hostname → label key → label value.
type ServerLabels = map[string]map[string]string

// Store defines the interface for user-defined label operations.
type Store interface {
	GetAll(cluster string) (ServerLabels, error)
	GetForServer(cluster, hostname string) (map[string]string, error)
	Set(cluster, hostname, key, value string) error
	Delete(cluster, hostname, key string) error
	MergeAll(cluster string, labels ServerLabels) error
	ReplaceAll(cluster string, labels ServerLabels) error
}

// FileStore implements Store using the local filesystem.
type FileStore struct {
	BaseDir string
}

// NewFileStore creates a FileStore at the default XDG data location.
func NewFileStore() (*FileStore, error) {
	base, err := DefaultBaseDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{BaseDir: base}, nil
}

// DefaultBaseDir returns ~/.local/share/teash (or XDG_DATA_HOME/teash).
func DefaultBaseDir() (string, error) {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "teash"), nil
}

func (fs *FileStore) path(cluster string) string {
	return filepath.Join(fs.BaseDir, sanitizeFilename(cluster)+"-labels.json")
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(name)
}

func (fs *FileStore) load(cluster string) (ServerLabels, error) {
	data, err := os.ReadFile(fs.path(cluster))
	if err != nil {
		if os.IsNotExist(err) {
			return make(ServerLabels), nil
		}
		return nil, fmt.Errorf("reading labels for %q: %w", cluster, err)
	}
	var labels ServerLabels
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, fmt.Errorf("parsing labels for %q: %w", cluster, err)
	}
	if labels == nil {
		labels = make(ServerLabels)
	}
	return labels, nil
}

func (fs *FileStore) save(cluster string, labels ServerLabels) error {
	if err := os.MkdirAll(fs.BaseDir, 0o755); err != nil {
		return fmt.Errorf("creating labels directory: %w", err)
	}
	data, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling labels: %w", err)
	}
	if err := os.WriteFile(fs.path(cluster), data, 0o644); err != nil {
		return fmt.Errorf("writing labels for %q: %w", cluster, err)
	}
	return nil
}

func (fs *FileStore) GetAll(cluster string) (ServerLabels, error) {
	return fs.load(cluster)
}

func (fs *FileStore) GetForServer(cluster, hostname string) (map[string]string, error) {
	labels, err := fs.load(cluster)
	if err != nil {
		return nil, err
	}
	if sl, ok := labels[hostname]; ok {
		return sl, nil
	}
	return nil, nil
}

func (fs *FileStore) Set(cluster, hostname, key, value string) error {
	labels, err := fs.load(cluster)
	if err != nil {
		return err
	}
	if labels[hostname] == nil {
		labels[hostname] = make(map[string]string)
	}
	labels[hostname][key] = value
	return fs.save(cluster, labels)
}

func (fs *FileStore) Delete(cluster, hostname, key string) error {
	labels, err := fs.load(cluster)
	if err != nil {
		return err
	}
	if labels[hostname] != nil {
		delete(labels[hostname], key)
		if len(labels[hostname]) == 0 {
			delete(labels, hostname)
		}
	}
	return fs.save(cluster, labels)
}

func (fs *FileStore) MergeAll(cluster string, incoming ServerLabels) error {
	labels, err := fs.load(cluster)
	if err != nil {
		return err
	}
	for host, kvs := range incoming {
		if labels[host] == nil {
			labels[host] = make(map[string]string)
		}
		for k, v := range kvs {
			labels[host][k] = v
		}
	}
	return fs.save(cluster, labels)
}

func (fs *FileStore) ReplaceAll(cluster string, labels ServerLabels) error {
	return fs.save(cluster, labels)
}
