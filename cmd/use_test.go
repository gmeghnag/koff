package koff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gmeghnag/koff/types"
)

func TestUseContextWritesConfig(t *testing.T) {
	dir := t.TempDir()
	// Point the config at a not-yet-existing .koff subdirectory to ensure
	// useContext creates the parent directory.
	cfg := filepath.Join(dir, ".koff", "koff.json")
	snap := filepath.Join(dir, "snapshot.db")
	if err := os.WriteFile(snap, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := useContext(snap, cfg); err != nil {
		t.Fatalf("useContext: %v", err)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	var c types.Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("config is not valid JSON: %v", err)
	}
	if c.InUse.Path != snap {
		t.Errorf("InUse.Path = %q, want %q", c.InUse.Path, snap)
	}
	if !c.InUse.IsEtcdDb {
		t.Error("expected IsEtcdDb=true for a .db file")
	}
}

func TestUseContextNonDBFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".koff", "koff.json")
	res := filepath.Join(dir, "resources.yaml")
	if err := os.WriteFile(res, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := useContext(res, cfg); err != nil {
		t.Fatalf("useContext: %v", err)
	}
	data, _ := os.ReadFile(cfg)
	var c types.Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	if c.InUse.IsEtcdDb {
		t.Error("expected IsEtcdDb=false for a .yaml file")
	}
}
