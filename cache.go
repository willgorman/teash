package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type cachedNodes struct {
	Nodes      Nodes     `yaml:"nodes"`
	UpdateTime time.Time `yaml:"update_time"`
}

type cachingTeleport struct {
	wrapped Teleport
	cache   cachedNodes
}

func NewCachingTeleport(t Teleport) Teleport {
	return &cachingTeleport{
		wrapped: t,
	}
}

func (c *cachingTeleport) getCachePath() (string, error) {
	cluster, err := c.wrapped.GetCluster()
	if err != nil {
		return "", err
	}

	configDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(configDir, ".config", "teash")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, fmt.Sprintf("%s-nodes.yaml", cluster)), nil
}

func (c *cachingTeleport) loadCache() error {
	path, err := c.getCachePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return yaml.Unmarshal(data, &c.cache)
}

func (c *cachingTeleport) saveCache() error {
	path, err := c.getCachePath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c.cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (c *cachingTeleport) GetNodes(refresh bool) (Nodes, error) {
	if refresh {
		nodes, err := c.wrapped.GetNodes(true)
		if err != nil {
			return nil, err
		}
		c.cache = cachedNodes{
			Nodes:      nodes,
			UpdateTime: time.Now(),
		}
		if err := c.saveCache(); err != nil {
			return nil, err
		}
		return nodes, nil
	}

	if err := c.loadCache(); err != nil {
		return nil, err
	}

	if len(c.cache.Nodes) > 0 {
		return c.cache.Nodes, nil
	}

	return c.GetNodes(true)
}

func (c *cachingTeleport) GetCluster() (string, error) {
	return c.wrapped.GetCluster()
}

func (c *cachingTeleport) Connect(cmd []string) {
	c.wrapped.Connect(cmd)
}
