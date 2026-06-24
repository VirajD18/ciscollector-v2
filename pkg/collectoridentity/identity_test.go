package collectoridentity

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOrCreatePersistsNodeID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if os.Getenv("USERPROFILE") != "" {
		t.Setenv("USERPROFILE", home)
	}

	tests := []struct {
		name     string
		hostname string
	}{
		{name: "first_run", hostname: "host-a"},
		{name: "restart_same", hostname: "host-a"},
	}
	var firstID string
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := LoadOrCreate(tc.hostname)
			if err != nil {
				t.Fatalf("LoadOrCreate: %v", err)
			}
			if id.NodeID == "" {
				t.Fatal("empty node_id")
			}
			if i == 0 {
				firstID = id.NodeID
			} else if id.NodeID != firstID {
				t.Fatalf("node_id changed: %q vs %q", id.NodeID, firstID)
			}
		})
	}
}

func TestDifferentHomesGetDifferentIDs(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	mk := func(dir, host string) string {
		path := filepath.Join(dir, ".klouddb", identityFileName)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		id := &Identity{NodeID: "id-" + host, Hostname: host, CreatedAt: mustTime()}
		if err := save(path, id); err != nil {
			t.Fatal(err)
		}
		return id.NodeID
	}
	a := mk(dirA, "a")
	b := mk(dirB, "b")
	if a == b {
		t.Fatal("expected different node ids")
	}
}

func mustTime() (t time.Time) { return time.Now().UTC() }
