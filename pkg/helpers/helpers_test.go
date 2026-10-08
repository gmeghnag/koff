package helpers

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/gmeghnag/koff/types"
)

// getArgsKeys returns the sorted resource-type keys stored in Koff.GetArgs.
func getArgsKeys(k *types.KoffCommand) []string {
	keys := make([]string, 0, len(k.GetArgs))
	for key := range k.GetArgs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// names returns the sorted set of resource names requested for a given type.
func names(k *types.KoffCommand, resourceType string) []string {
	out := make([]string, 0)
	for n := range k.GetArgs[resourceType] {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func TestParseGetArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantTypes   []string
		wantNames   map[string][]string
		wantSingle  bool
		wantUnknown []string // keys expected in ArgPresent marked false (not resolvable)
	}{
		{
			name:      "single type plural",
			args:      []string{"pods"},
			wantTypes: []string{"pod"},
			wantNames: map[string][]string{"pod": {}},
		},
		{
			name:      "short alias",
			args:      []string{"po"},
			wantTypes: []string{"pod"},
			wantNames: map[string][]string{"pod": {}},
		},
		{
			name:      "comma separated types",
			args:      []string{"po,svc"},
			wantTypes: []string{"pod", "service"},
			wantNames: map[string][]string{"pod": {}, "service": {}},
		},
		{
			name:       "slash form single",
			args:       []string{"pod/foo"},
			wantTypes:  []string{"pod"},
			wantNames:  map[string][]string{"pod": {"foo"}},
			wantSingle: true,
		},
		{
			name:      "slash form multiple",
			args:      []string{"pod/foo", "svc/bar"},
			wantTypes: []string{"pod", "service"},
			wantNames: map[string][]string{"pod": {"foo"}, "service": {"bar"}},
		},
		{
			name:       "type then names",
			args:       []string{"pod", "foo"},
			wantTypes:  []string{"pod"},
			wantNames:  map[string][]string{"pod": {"foo"}},
			wantSingle: true,
		},
		{
			name:      "type then multiple names",
			args:      []string{"pod", "foo", "bar"},
			wantTypes: []string{"pod"},
			wantNames: map[string][]string{"pod": {"bar", "foo"}},
		},
		{
			// Regression for the SplitN(..., ".", 1) bug: the group suffix must be
			// stripped so the resource resolves to its kind ("deployment").
			name:      "group qualified type then names",
			args:      []string{"deployment.apps", "foo", "bar"},
			wantTypes: []string{"deployment"},
			wantNames: map[string][]string{"deployment": {"bar", "foo"}},
		},
		{
			name:      "group qualified single type",
			args:      []string{"deployment.apps"},
			wantTypes: []string{"deployment"},
			wantNames: map[string][]string{"deployment": {}},
		},
		{
			name:       "group qualified slash form",
			args:       []string{"deployment.apps/foo"},
			wantTypes:  []string{"deployment"},
			wantNames:  map[string][]string{"deployment": {"foo"}},
			wantSingle: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			koff := types.NewKoffCommand()
			if err := ParseGetArgs(koff, tc.args); err != nil {
				t.Fatalf("ParseGetArgs returned error: %v", err)
			}
			if got := getArgsKeys(koff); !reflect.DeepEqual(got, tc.wantTypes) {
				t.Errorf("resource types = %v, want %v", got, tc.wantTypes)
			}
			for rt, want := range tc.wantNames {
				if got := names(koff, rt); !reflect.DeepEqual(got, want) {
					t.Errorf("names for %q = %v, want %v", rt, got, want)
				}
			}
			if koff.SingleResource != tc.wantSingle {
				t.Errorf("SingleResource = %v, want %v", koff.SingleResource, tc.wantSingle)
			}
			for _, u := range tc.wantUnknown {
				if present, ok := koff.ArgPresent[u]; !ok || present {
					t.Errorf("expected ArgPresent[%q]=false, got ok=%v present=%v", u, ok, present)
				}
			}
		})
	}
}

func TestParseGetArgsMixedFormsRejected(t *testing.T) {
	koff := types.NewKoffCommand()
	// "resource name" + "resource/name" mixed is invalid.
	err := ParseGetArgs(koff, []string{"pod", "svc/bar"})
	if err == nil {
		t.Fatalf("expected error for mixed arg forms, got nil")
	}
}

func TestEtcdPrefixFromAliasCoreOverrides(t *testing.T) {
	koff := types.NewKoffCommand()
	koff.IsEtcdDb = true
	koff.AllNamespaces = true
	cases := map[string]string{
		"nodes":     "/kubernetes.io/minions",
		"node":      "/kubernetes.io/minions",
		"no":        "/kubernetes.io/minions",
		"services":  "/kubernetes.io/services/specs",
		"endpoints": "/kubernetes.io/services/endpoints",
		"pods":      "/kubernetes.io/pods",
	}
	for alias, want := range cases {
		normalized, err := normalizeResourceAlias(koff, alias)
		if err != nil {
			t.Fatalf("normalizeResourceAlias(%q): %v", alias, err)
		}
		got, err := EtcdPrefixFromAlias(koff, normalized, "")
		if err != nil {
			t.Fatalf("EtcdPrefixFromAlias(%q): %v", alias, err)
		}
		if got != want {
			t.Errorf("alias %q -> prefix %q, want %q", alias, got, want)
		}
	}
}

func TestRetrieveKindGroupFromCRDSCaching(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	crdDir := filepath.Join(home, ".koff", "customresourcedefinitions")
	if err := os.MkdirAll(crdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	crdYAML := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.example.com
spec:
  group: example.com
  names:
    kind: Widget
    plural: widgets
    singular: widget
    shortNames: [wg]
  scope: Namespaced
`
	if err := os.WriteFile(filepath.Join(crdDir, "widgets.yaml"), []byte(crdYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	koff := types.NewKoffCommand()
	for _, alias := range []string{"widget", "widgets", "wg", "widget.example.com"} {
		kind, group, err := RetrieveKindGroupFromCRDS(koff, alias)
		if err != nil {
			t.Fatalf("alias %q: %v", alias, err)
		}
		if kind != "widget" || group != "example.com" {
			t.Errorf("alias %q -> (%q,%q), want (widget,example.com)", alias, kind, group)
		}
	}
	if !koff.CRDsLoaded {
		t.Error("expected CRDsLoaded=true after first lookup")
	}

	// Remove the directory: a cached lookup must still resolve (proving no re-read).
	os.RemoveAll(crdDir)
	if kind, _, err := RetrieveKindGroupFromCRDS(koff, "widget"); err != nil || kind != "widget" {
		t.Errorf("cached lookup failed after dir removal: kind=%q err=%v", kind, err)
	}
}

func TestRetrieveKindGroupFromCRDSUnknown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	koff := types.NewKoffCommand()
	if _, _, err := RetrieveKindGroupFromCRDS(koff, "nonexistent"); err == nil {
		t.Error("expected error for unknown alias with no CRDs")
	}
}
