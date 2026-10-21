package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Action    string   `yaml:"action"`
	Direction string   `yaml:"direction"`
	To        string   `yaml:"to,omitempty"`
	From      string   `yaml:"from,omitempty"`
	Ports     []int    `yaml:"ports,omitempty"`
}

type Scope struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Subnet      string   `yaml:"subnet"`
	Policies    []Policy `yaml:"policies"`
}

type IntentService struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
	Port   int    `yaml:"port"`
	Scope  string `yaml:"scope"`
}

type Peer struct {
	ID         string   `yaml:"id"`
	Endpoint   string   `yaml:"endpoint"`
	AllowedIPs []string `yaml:"allowed_ips"`
}

type TransportConfig struct {
	ListenPort int    `yaml:"listen_port"`
	Peers      []Peer `yaml:"peers"`
}

type NetworkConfig struct {
	OverlayInterface string          `yaml:"overlay_interface"`
	Subnet           string          `yaml:"subnet"`
	Transport        TransportConfig `yaml:"transport"`
}

type Manifest struct {
	Version string `yaml:"version"`
	Node    struct {
		ID    string `yaml:"id"`
		Scope string `yaml:"scope"`
	} `yaml:"node"`
	Network  NetworkConfig   `yaml:"network"`
	Scopes   []Scope         `yaml:"scopes"`
	Services []IntentService `yaml:"intent_services"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo manifiesto: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("error parseando YAML: %w", err)
	}

	return &m, nil
}
