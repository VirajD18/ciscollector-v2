package collectoridentity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const identityFileName = "collector-node.yaml"

// Identity is the stable collector agent identity persisted on disk.
type Identity struct {
	NodeID    string    `yaml:"node_id"`
	Hostname  string    `yaml:"hostname"`
	CreatedAt time.Time `yaml:"created_at"`
}

// LoadOrCreate returns persisted identity or creates a new one.
func LoadOrCreate(hostname string) (*Identity, error) {
	host := strings.TrimSpace(hostname)
	if host == "" {
		return nil, fmt.Errorf("hostname is required for collector identity")
	}
	path, err := identityPath()
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(path); err == nil {
		var id Identity
		if err := yaml.Unmarshal(data, &id); err != nil {
			return nil, fmt.Errorf("parse collector identity: %w", err)
		}
		id.NodeID = strings.TrimSpace(id.NodeID)
		if id.NodeID == "" {
			return nil, fmt.Errorf("collector identity file missing node_id")
		}
		if strings.TrimSpace(id.Hostname) != host {
			fmt.Printf("> Note: [app].hostname changed (%q -> %q); keeping node_id %s\n", id.Hostname, host, id.NodeID)
		}
		id.Hostname = host
		return &id, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	id := &Identity{
		NodeID:    uuid.NewString(),
		Hostname:  host,
		CreatedAt: time.Now().UTC(),
	}
	if err := save(path, id); err != nil {
		return nil, err
	}
	return id, nil
}

func identityPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".klouddb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, identityFileName), nil
}

func save(path string, id *Identity) error {
	b, err := yaml.Marshal(id)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
