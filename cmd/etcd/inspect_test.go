package etcd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// findSnapshot locates a local etcd snapshot under testdata/ (gitignored). The
// tests that need it are skipped when no snapshot is present so CI stays green.
func findSnapshot(t *testing.T) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", "*.db"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(matches) == 0 {
		t.Skip("no etcd snapshot in testdata/*.db; skipping")
	}
	return matches[0]
}

func TestInspectEtcdListKeys(t *testing.T) {
	snap := findSnapshot(t)
	var buf bytes.Buffer
	if err := inspectEtcd([]string{snap}, &buf); err != nil {
		t.Fatalf("inspectEtcd list: %v", err)
	}
	out := buf.String()
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected a non-empty key listing")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 10 {
		t.Fatalf("expected many keys, got %d", len(lines))
	}
	sample := lines
	if len(sample) > 5 {
		sample = sample[:5]
	}
	for _, l := range sample {
		if !strings.HasPrefix(l, "/") {
			t.Errorf("etcd key %q does not start with '/'", l)
		}
	}
}

func TestInspectEtcdGetKeyJSON(t *testing.T) {
	snap := findSnapshot(t)

	// First list keys and pick one to fetch.
	var list bytes.Buffer
	if err := inspectEtcd([]string{snap}, &list); err != nil {
		t.Fatalf("inspectEtcd list: %v", err)
	}
	key := strings.SplitN(strings.TrimSpace(list.String()), "\n", 2)[0]

	oldFormat := formatOutput
	formatOutput = "json"
	defer func() { formatOutput = oldFormat }()

	var buf bytes.Buffer
	if err := inspectEtcd([]string{snap, key}, &buf); err != nil {
		t.Fatalf("inspectEtcd get %q: %v", key, err)
	}

	var obj map[string]any
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatalf("output for %q is not valid JSON: %v\n%s", key, err, buf.String())
	}
	if _, ok := obj["kind"]; !ok {
		t.Errorf("decoded object for %q has no 'kind' field", key)
	}
}

func TestInspectEtcdGetKeyYAML(t *testing.T) {
	snap := findSnapshot(t)

	var list bytes.Buffer
	if err := inspectEtcd([]string{snap}, &list); err != nil {
		t.Fatalf("inspectEtcd list: %v", err)
	}
	key := strings.SplitN(strings.TrimSpace(list.String()), "\n", 2)[0]

	oldFormat := formatOutput
	formatOutput = "yaml"
	defer func() { formatOutput = oldFormat }()

	var buf bytes.Buffer
	if err := inspectEtcd([]string{snap, key}, &buf); err != nil {
		t.Fatalf("inspectEtcd get yaml %q: %v", key, err)
	}
	if !strings.Contains(buf.String(), "kind:") {
		t.Errorf("yaml output for %q missing 'kind:'\n%s", key, buf.String())
	}
}

func TestInspectEtcdKeyNotFound(t *testing.T) {
	snap := findSnapshot(t)
	var buf bytes.Buffer
	err := inspectEtcd([]string{snap, "/this/key/does/not/exist"}, &buf)
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestInspectEtcdMissingFile(t *testing.T) {
	var buf bytes.Buffer
	if err := inspectEtcd([]string{"/no/such/snapshot.db"}, &buf); err == nil {
		t.Fatal("expected error for missing snapshot file, got nil")
	}
}
