package manifest

import (
	"os"

	"gopkg.in/yaml.v3"
)

type IntentManifest struct {
	Version      string        `yaml:"version"`
	Metadata     Metadata      `yaml:"metadata"`
	ScopeContext ScopeContext  `yaml:"scope_context"`
	Network      NetworkConfig `yaml:"network"`
	Services     []Service     `yaml:"intent_services"`
}

type Metadata struct {
	NodeID     string `yaml:"node_id"`
	Federation string `yaml:"federation"`
}

type ScopeContext struct {
	ActiveScope string            `yaml:"active_scope"`
	Triggers    map[string]string `yaml:"triggers"`
}

type NetworkConfig struct {
	OverlaySubnet string `yaml:"overlay_subnet"`
	IsolationMode string `yaml:"isolation_mode"`
}

type Service struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
	Port   int    `yaml:"port"`
}

func LoadManifest(path string) (*IntentManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest IntentManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}
