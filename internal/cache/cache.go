package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry represents a cached set of server data for a cluster.
type Entry struct {
	Cluster   string          `json:"cluster"`
	UpdatedAt time.Time       `json:"updated_at"`
	RawNodes  json.RawMessage `json:"raw_nodes"`
}

// Store defines the interface for cache operations.
type Store interface {
	Get(cluster string) (*Entry, error)
	Put(entry *Entry) error
	Delete(cluster string) error
	LastUpdated(cluster string) (time.Time, error)
}

// FileStore implements Store using the local filesystem.
type FileStore struct {
	BaseDir string
}

// NewFileStore creates a FileStore at the default XDG cache location.
func NewFileStore() (*FileStore, error) {
	base, err := DefaultBaseDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{BaseDir: base}, nil
}

// DefaultBaseDir returns ~/.cache/teash (or XDG_CACHE_HOME/teash).
func DefaultBaseDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("determining cache directory: %w", err)
	}
	return filepath.Join(cacheDir, "teash"), nil
}

func (fs *FileStore) path(cluster string) string {
	return filepath.Join(fs.BaseDir, sanitizeFilename(cluster)+".json")
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(name)
}

func (fs *FileStore) Get(cluster string) (*Entry, error) {
	data, err := os.ReadFile(fs.path(cluster))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache for %q: %w", cluster, err)
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Treat corrupted cache as empty
		log.Printf("warning: corrupt cache for %q, treating as empty: %v", cluster, err)
		return nil, nil
	}
	return &entry, nil
}

func (fs *FileStore) Put(entry *Entry) error {
	if err := os.MkdirAll(fs.BaseDir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling cache entry: %w", err)
	}
	if err := os.WriteFile(fs.path(entry.Cluster), data, 0o644); err != nil {
		return fmt.Errorf("writing cache for %q: %w", entry.Cluster, err)
	}
	return nil
}

func (fs *FileStore) Delete(cluster string) error {
	err := os.Remove(fs.path(cluster))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting cache for %q: %w", cluster, err)
	}
	return nil
}

func (fs *FileStore) LastUpdated(cluster string) (time.Time, error) {
	entry, err := fs.Get(cluster)
	if err != nil {
		return time.Time{}, err
	}
	if entry == nil {
		return time.Time{}, nil
	}
	return entry.UpdatedAt, nil
}
