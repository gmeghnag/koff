package get

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gmeghnag/koff/types"
	bolt "go.etcd.io/bbolt"
)

// openSnapshot opens the local etcd snapshot (gitignored) read-only, skipping
// the test when none is available so CI stays green.
func openSnapshot(t testing.TB) *types.KoffCommand {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join("..", "..", "testdata", "*.db"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(matches) == 0 {
		t.Skip("no etcd snapshot in testdata/*.db; skipping")
	}
	koff := types.NewKoffCommand()
	koff.IsEtcdDb = true
	db, err := bolt.Open(matches[0], 0400, &bolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	koff.EtcdDb = db
	return koff
}

func TestGetFromEtcdNamespaces(t *testing.T) {
	koff := openSnapshot(t)
	koff.AllNamespaces = true
	koff.OutputFormat = "name"

	if err := GetFromEtcd(koff, []string{"namespaces"}); err != nil {
		t.Fatalf("GetFromEtcd namespaces: %v", err)
	}
	out := koff.Output.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("expected at least one namespace")
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "namespace/") {
			t.Errorf("unexpected line %q, want namespace/ prefix", l)
		}
	}
}

func TestGetFromEtcdUnknownResource(t *testing.T) {
	koff := openSnapshot(t)
	err := GetFromEtcd(koff, []string{"thisisnotarealresource"})
	if err == nil {
		t.Fatal("expected error for unknown resource type, got nil")
	}
}

func TestGetFromEtcdPodsAllNamespacesYAML(t *testing.T) {
	koff := openSnapshot(t)
	koff.AllNamespaces = true
	koff.OutputFormat = "yaml"

	if err := GetFromEtcd(koff, []string{"pods"}); err != nil {
		t.Fatalf("GetFromEtcd pods: %v", err)
	}
	// We can't assume the cluster has pods, but if items exist they must be Pods.
	for _, item := range koff.UnstructuredList.Items {
		if item.GetKind() != "Pod" {
			t.Errorf("got non-Pod item kind %q", item.GetKind())
		}
	}
}

// discoverCustomResource finds a (group, plural) pair for a custom resource
// that has at least one instance in the snapshot, by inspecting the key layout
// /kubernetes.io/<group-with-dot>/<plural>/...
func discoverCustomResource(t *testing.T, koff *types.KoffCommand) (group, plural string) {
	t.Helper()
	if err := buildEtcdKeyIndex(koff); err != nil {
		t.Fatalf("buildEtcdKeyIndex: %v", err)
	}
	for kubeKey := range koff.KubeKeysToEtcdKeys {
		parts := strings.Split(kubeKey, "/")
		// ["", "kubernetes.io", "<group>", "<plural>", ...]
		if len(parts) >= 5 && strings.Contains(parts[2], ".") &&
			parts[2] != "apiextensions.k8s.io" && parts[3] != "" {
			return parts[2], parts[3]
		}
	}
	t.Skip("no custom resource instances found in snapshot")
	return "", ""
}

// TestGetFromEtcdCustomResource verifies the lazy CRD-loading path: requesting a
// custom resource must trigger CRD resolution and return its instances.
func TestGetFromEtcdCustomResource(t *testing.T) {
	koff := openSnapshot(t)
	group, plural := discoverCustomResource(t, koff)

	koff.AllNamespaces = true
	koff.OutputFormat = "name"
	arg := plural + "." + group
	if err := GetFromEtcd(koff, []string{arg}); err != nil {
		t.Fatalf("GetFromEtcd %q: %v", arg, err)
	}
	if strings.TrimSpace(koff.Output.String()) == "" {
		t.Errorf("expected at least one %q instance in output", arg)
	}
	if len(koff.EtcdAliasToCrdKubeKey) == 0 {
		t.Error("expected CRD aliases to have been loaded lazily")
	}
}

// TestGetFromEtcdDeterministicOrder ensures the parallel decode path still
// produces a stable, sorted output across repeated runs.
func TestGetFromEtcdDeterministicOrder(t *testing.T) {
	run := func() string {
		koff := openSnapshot(t)
		koff.AllNamespaces = true
		koff.OutputFormat = "name"
		if err := GetFromEtcd(koff, []string{"pods"}); err != nil {
			t.Fatalf("GetFromEtcd: %v", err)
		}
		return koff.Output.String()
	}
	first := run()
	for i := 0; i < 3; i++ {
		if got := run(); got != first {
			t.Fatalf("non-deterministic output on run %d", i)
		}
	}
}

func BenchmarkGetFromEtcdAllPods(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		koff := openSnapshot(b)
		koff.AllNamespaces = true
		koff.OutputFormat = "name"
		b.StartTimer()
		if err := GetFromEtcd(koff, []string{"pods"}); err != nil {
			b.Fatalf("GetFromEtcd: %v", err)
		}
	}
}

// TestGetFromEtcdNodes is a regression test: nodes are stored under the legacy
// "minions" etcd key, so "koff get nodes" must still find them.
func TestGetFromEtcdNodes(t *testing.T) {
	koff := openSnapshot(t)
	koff.OutputFormat = "name"
	if err := GetFromEtcd(koff, []string{"nodes"}); err != nil {
		t.Fatalf("GetFromEtcd nodes: %v", err)
	}
	out := strings.TrimSpace(koff.Output.String())
	if out == "" {
		t.Fatal("expected at least one node, got none")
	}
	for _, l := range strings.Split(out, "\n") {
		if !strings.HasPrefix(l, "node/") {
			t.Errorf("unexpected line %q, want node/ prefix", l)
		}
	}
}

// TestGetFromEtcdNodesTable renders nodes with the default (table) output to
// ensure the Node table path does not panic on snapshot data.
func TestGetFromEtcdNodesTable(t *testing.T) {
	koff := openSnapshot(t)
	if err := GetFromEtcd(koff, []string{"nodes"}); err != nil {
		t.Fatalf("GetFromEtcd nodes (table): %v", err)
	}
	var buf bytes.Buffer
	if err := KoffToWriter(koff, &buf); err != nil {
		t.Fatalf("KoffToWriter: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("node/")) {
		t.Errorf("expected node rows in table output:\n%s", buf.String())
	}
}

func BenchmarkGetFromEtcdAllPodsTable(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		koff := openSnapshot(b)
		koff.AllNamespaces = true
		koff.OutputFormat = "" // default: table
		b.StartTimer()
		if err := GetFromEtcd(koff, []string{"pods"}); err != nil {
			b.Fatalf("GetFromEtcd: %v", err)
		}
	}
}

// TestGetFromEtcdTableParallelMatchesSerial renders the same table query with
// the parallel path and with GOMAXPROCS(1) (which forces the serial builder),
// and asserts byte-identical output — guarding against ordering/races.
func TestGetFromEtcdTableParallelMatchesSerial(t *testing.T) {
	render := func() string {
		koff := openSnapshot(t)
		koff.AllNamespaces = true // default (table) output
		if err := GetFromEtcd(koff, []string{"pods"}); err != nil {
			t.Fatalf("GetFromEtcd: %v", err)
		}
		var buf bytes.Buffer
		if err := KoffToWriter(koff, &buf); err != nil {
			t.Fatalf("KoffToWriter: %v", err)
		}
		return buf.String()
	}
	old := runtime.GOMAXPROCS(1)
	serial := render()
	runtime.GOMAXPROCS(old)
	parallel := render()
	if serial != parallel {
		t.Errorf("parallel table output differs from serial:\n--serial--\n%s\n--parallel--\n%s", serial, parallel)
	}
}

// TestGetFromEtcdTableRowCountMatchesNames ensures the table renders exactly the
// same set of objects as -o name (no drops or duplicates from parallelism).
func TestGetFromEtcdTableRowCountMatchesNames(t *testing.T) {
	nameKoff := openSnapshot(t)
	nameKoff.AllNamespaces = true
	nameKoff.OutputFormat = "name"
	if err := GetFromEtcd(nameKoff, []string{"pods"}); err != nil {
		t.Fatal(err)
	}
	names := strings.Count(strings.TrimSpace(nameKoff.Output.String()), "\n") + 1

	tableKoff := openSnapshot(t)
	tableKoff.AllNamespaces = true
	if err := GetFromEtcd(tableKoff, []string{"pods"}); err != nil {
		t.Fatal(err)
	}
	// Each row is a pod; header lines are not counted in Table.Rows.
	rows := len(tableKoff.Table.Rows)
	if rows != names {
		t.Errorf("table rows = %d, name lines = %d (should match)", rows, names)
	}
}
