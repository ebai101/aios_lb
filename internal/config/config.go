// Package config
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Debug     bool            `yaml:"debug"`
	Instances []InstanceGroup `yaml:"instances"`
}

type InstanceGroup struct {
	Type string   `yaml:"type"`
	URLs []string `yaml:"urls"`
}

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

func (c *Config) ToRouteMap() map[string][]string {
	routes := make(map[string][]string)
	for _, group := range c.Instances {
		routes[group.Type] = group.URLs
	}
	return routes
}
