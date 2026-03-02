// Package config
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the root of the config.yaml file.
type Config struct {
	Debug     bool            `yaml:"debug"`
	Instances []InstanceGroup `yaml:"instances"`
}

// InstanceGroup represents a single addon type and its upstream URLs.
type InstanceGroup struct {
	Type string   `yaml:"type"`
	URLs []string `yaml:"urls"`
}

// Load reads and unmarshals the YAML configuration from the given file path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ToRouteMap converts the YAML struct slice into a fast O(1) lookup map.
// This matches the format expected by our ProxyHandler.
func (c *Config) ToRouteMap() map[string][]string {
	routes := make(map[string][]string)
	for _, group := range c.Instances {
		routes[group.Type] = group.URLs
	}
	return routes
}
