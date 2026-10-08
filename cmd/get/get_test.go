package get

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func mustUnstructured(t *testing.T, manifest string) unstructured.Unstructured {
	t.Helper()
	obj := unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(manifest), &obj.Object); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}
	return obj
}

const podManifest = `
apiVersion: v1
kind: Pod
metadata:
  name: mypod
  namespace: demo
  creationTimestamp: "2020-01-01T00:00:00Z"
status:
  phase: Running
  containerStatuses:
  - ready: true
    restartCount: 0
spec:
  containers:
  - name: c
    image: busybox
`

// TestHandleObjectKnownResourceBuildsTable verifies a known resource (Pod) is
// rendered through the internal table generator without panicking and that the
// Name cell is formatted as kind/name.
func TestHandleObjectKnownResourceBuildsTable(t *testing.T) {
	koff := types.NewKoffCommand()
	if err := helpers.ParseGetArgs(koff, []string{"pods"}); err != nil {
		t.Fatalf("ParseGetArgs: %v", err)
	}
	koff.FromInput = true

	pod := mustUnstructured(t, podManifest)
	if err := HandleObject(koff, pod); err != nil {
		t.Fatalf("HandleObject returned error: %v", err)
	}

	if koff.CurrentKind != "Pod" {
		t.Fatalf("CurrentKind = %q, want Pod", koff.CurrentKind)
	}
	if len(koff.Table.Rows) != 1 {
		t.Fatalf("Table.Rows = %d, want 1", len(koff.Table.Rows))
	}
	name, ok := koff.Table.Rows[0].Cells[0].(string)
	if !ok {
		t.Fatalf("name cell is not a string: %T", koff.Table.Rows[0].Cells[0])
	}
	if name != "pod/mypod" {
		t.Errorf("name cell = %q, want pod/mypod", name)
	}
}

// TestHandleObjectNameOutput verifies -o name output for core and grouped kinds.
func TestHandleObjectNameOutput(t *testing.T) {
	koff := types.NewKoffCommand()
	koff.OutputFormat = "name"
	koff.FromInput = true

	pod := mustUnstructured(t, podManifest)
	if err := HandleObject(koff, pod); err != nil {
		t.Fatalf("HandleObject: %v", err)
	}
	if got := koff.Output.String(); got != "pod/mypod\n" {
		t.Errorf("name output = %q, want %q", got, "pod/mypod\n")
	}
}

// TestHandleObjectNamespaceFilter verifies namespace filtering skips objects in
// other namespaces while keeping matches.
func TestHandleObjectNamespaceFilter(t *testing.T) {
	koff := types.NewKoffCommand()
	koff.OutputFormat = "name"
	koff.FromInput = true
	koff.Namespace = "other"

	pod := mustUnstructured(t, podManifest) // namespace: demo
	if err := HandleObject(koff, pod); err != nil {
		t.Fatalf("HandleObject: %v", err)
	}
	if got := koff.Output.String(); strings.TrimSpace(got) != "" {
		t.Errorf("expected pod in namespace demo to be filtered out, got %q", got)
	}
}

func TestKoffToWriterTable(t *testing.T) {
	koff := types.NewKoffCommand()
	if err := helpers.ParseGetArgs(koff, []string{"pods"}); err != nil {
		t.Fatal(err)
	}
	koff.FromInput = true
	if err := HandleObject(koff, mustUnstructured(t, podManifest)); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := KoffToWriter(koff, &buf); err != nil {
		t.Fatalf("KoffToWriter: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "pod/mypod") {
		t.Errorf("table output missing header/row:\n%s", out)
	}
}

func TestKoffToWriterJSONSingle(t *testing.T) {
	koff := types.NewKoffCommand()
	koff.FromInput = true
	koff.OutputFormat = "json"
	koff.SingleResource = true
	if err := HandleObject(koff, mustUnstructured(t, podManifest)); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := KoffToWriter(koff, &buf); err != nil {
		t.Fatalf("KoffToWriter: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, buf.String())
	}
	if obj["kind"] != "Pod" {
		t.Errorf("kind = %v, want Pod", obj["kind"])
	}
}

func TestKoffToWriterNoResources(t *testing.T) {
	koff := types.NewKoffCommand()
	var buf bytes.Buffer
	if err := KoffToWriter(koff, &buf); err != nil {
		t.Fatalf("KoffToWriter: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "No resources found." {
		t.Errorf("got %q, want 'No resources found.'", buf.String())
	}
}

func TestKoffToWriterUnknownResourceErrors(t *testing.T) {
	koff := types.NewKoffCommand()
	koff.GetArgs["frobnicator"] = map[string]struct{}{}
	koff.ArgPresent["frobnicator"] = false
	var buf bytes.Buffer
	if err := KoffToWriter(koff, &buf); err == nil {
		t.Fatal("expected error for unknown resource type, got nil")
	}
}

func TestHandleDataInEmpty(t *testing.T) {
	koff := types.NewKoffCommand()
	koff.FromInput = true
	for _, in := range [][]byte{nil, {}, []byte("   \n"), []byte("---\n")} {
		if err := HandleDataIn(in, koff); err != nil {
			t.Errorf("HandleDataIn(%q) error: %v", in, err)
		}
	}
}
