package get

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/gmeghnag/koff/cmd/etcd"
	"github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

const JsonMediaType = "application/json"

// GetFromEtcd runs the full "get" pipeline against an already-open etcd
// snapshot (koff.EtcdDb): it loads CRD aliases, parses the requested resource
// arguments into etcd key prefixes, collects the matching kube keys and renders
// each resource. It assumes koff.EtcdDb is open and owned by the caller.
func GetFromEtcd(koff *types.KoffCommand, args []string) error {
	if err := buildEtcdKeyIndex(koff); err != nil {
		return err
	}
	// CRD definitions are only needed to resolve custom-resource aliases.
	// Parsing every CRD is expensive, so skip it entirely when the request
	// targets only built-in resources (the common case, e.g. "pods").
	if etcdRequestNeedsCRDs(koff, args) {
		if err := loadCRDsFromEtcd(koff); err != nil {
			return err
		}
	}
	if err := helpers.ParseGetArgs(koff, args); err != nil {
		return err
	}

	etcdKeyPrefixesToCheck := make(map[string]bool)
	for resourceType := range koff.GetArgs {
		if len(koff.GetArgs[resourceType]) == 0 {
			prefix, err := helpers.EtcdPrefixFromAlias(koff, resourceType, "")
			if err != nil {
				return err
			}
			etcdKeyPrefixesToCheck[prefix] = false
		} else {
			for resourceName := range koff.GetArgs[resourceType] {
				prefix, err := helpers.EtcdPrefixFromAlias(koff, resourceType, resourceName)
				if err != nil {
					return err
				}
				etcdKeyPrefixesToCheck[prefix] = false
			}
		}
	}

	if err := GetResourcesFromEtcd(koff, etcdKeyPrefixesToCheck); err != nil {
		return err
	}
	sort.Strings(koff.EtcdKubeKeysToGet)
	// Snapshots can contain duplicate kube keys (multiple revisions); the slice
	// is sorted above so duplicates are adjacent.
	kubeKeys := dedupeSorted(koff.EtcdKubeKeysToGet)
	return renderKubeKeys(koff, kubeKeys)
}

// dedupeSorted removes adjacent duplicates from a sorted slice, returning a new
// slice.
func dedupeSorted(sorted []string) []string {
	out := make([]string, 0, len(sorted))
	for i, s := range sorted {
		if i == 0 || sorted[i-1] != s {
			out = append(out, s)
		}
	}
	return out
}

// parallelDecodeThreshold is the number of resources above which decoding is
// parallelised across CPU cores. Below it the goroutine overhead is not worth
// it and a single pass is used.
const parallelDecodeThreshold = 16

// renderKubeKeys decodes the given (sorted, de-duplicated) kube keys into
// unstructured objects and renders them in order. Two CPU-bound, per-object
// stages are parallelised across workers: decoding and (for tabular output)
// table generation. The final merge into the shared table/output state runs
// serially, in order, so output stays deterministic and race-free.
func renderKubeKeys(koff *types.KoffCommand, kubeKeys []string) error {
	objects, err := decodeKubeKeys(koff, kubeKeys)
	if err != nil {
		return err
	}

	// For json/yaml/name the per-object work is cheap, so render serially.
	tabular := koff.OutputFormat != "json" && koff.OutputFormat != "yaml" && koff.OutputFormat != "name"
	if !tabular {
		for _, obj := range objects {
			if obj == nil {
				continue
			}
			if err := HandleObject(koff, *obj); err != nil {
				return err
			}
		}
		return nil
	}

	// Table output: building each object's table (marshal + decode + generate)
	// dominates the runtime and is independent per object, so do it in parallel
	// with read-only helpers, then merge serially in order.
	tables, err := buildTablesParallel(koff, objects)
	if err != nil {
		return err
	}
	for i, obj := range objects {
		if obj == nil {
			continue
		}
		if !passesFilters(koff, *obj) {
			continue
		}
		koff.LastKind = obj.GetKind()
		if err := mergeObjectTable(koff, *obj, tables[i]); err != nil {
			return err
		}
	}
	return nil
}

// buildTablesParallel computes each object's table concurrently using the
// read-only builder, preserving input order. Entries for nil objects are nil.
func buildTablesParallel(koff *types.KoffCommand, objects []*unstructured.Unstructured) ([]*metav1.Table, error) {
	tables := make([]*metav1.Table, len(objects))

	workers := runtime.GOMAXPROCS(0)
	if len(objects) < parallelDecodeThreshold || workers <= 1 {
		for i, obj := range objects {
			if obj == nil {
				continue
			}
			t, err := buildObjectTableReadOnly(koff, *obj)
			if err != nil {
				return nil, err
			}
			tables[i] = t
		}
		return tables, nil
	}

	if workers > len(objects) {
		workers = len(objects)
	}
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	stripe := (len(objects) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		lo := w * stripe
		if lo >= len(objects) {
			break
		}
		hi := lo + stripe
		if hi > len(objects) {
			hi = len(objects)
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			for i := lo; i < hi; i++ {
				if objects[i] == nil {
					continue
				}
				t, err := buildObjectTableReadOnly(koff, *objects[i])
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
				tables[i] = t
			}
		}(lo, hi)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return tables, nil
}

// decodeKubeKeys resolves each kube key to its stored value and decodes it into
// an unstructured object, preserving input order (nil entries are keys that
// could not be decoded and are skipped by the caller).
func decodeKubeKeys(koff *types.KoffCommand, kubeKeys []string) ([]*unstructured.Unstructured, error) {
	objects := make([]*unstructured.Unstructured, len(kubeKeys))

	workers := runtime.GOMAXPROCS(0)
	if len(kubeKeys) < parallelDecodeThreshold || workers <= 1 {
		if err := koff.EtcdDb.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("key"))
			if b == nil {
				return fmt.Errorf("bucket %q not found", "key")
			}
			for i, kubeKey := range kubeKeys {
				objects[i] = decodeKubeValue(koff, b, kubeKey)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return objects, nil
	}

	if workers > len(kubeKeys) {
		workers = len(kubeKeys)
	}
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	stripe := (len(kubeKeys) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		lo := w * stripe
		if lo >= len(kubeKeys) {
			break
		}
		hi := lo + stripe
		if hi > len(kubeKeys) {
			hi = len(kubeKeys)
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			// Each worker uses its own read-only transaction; bbolt supports
			// any number of concurrent readers.
			err := koff.EtcdDb.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte("key"))
				if b == nil {
					return fmt.Errorf("bucket %q not found", "key")
				}
				for i := lo; i < hi; i++ {
					objects[i] = decodeKubeValue(koff, b, kubeKeys[i])
				}
				return nil
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(lo, hi)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return objects, nil
}

// decodeKubeValue fetches and decodes a single stored value into an unstructured
// object, handling both JSON-stored records and Kubernetes protobuf. It returns
// nil for keys that are missing or cannot be decoded (mirroring the previous
// best-effort behaviour). It only reads shared state and is safe to call
// concurrently from multiple goroutines, each with its own bucket handle.
func decodeKubeValue(koff *types.KoffCommand, b *bolt.Bucket, kubeKey string) *unstructured.Unstructured {
	etcdKey, exists := koff.KubeKeysToEtcdKeys[kubeKey]
	if !exists {
		return nil
	}
	var kv mvccpb.KeyValue
	if err := kv.Unmarshal(b.Get(etcdKey)); err != nil {
		return nil
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(kv.Value); err == nil {
		return u
	}
	var buf bytes.Buffer
	if _, err := etcd.DetectAndConvert(JsonMediaType, kv.Value, &buf); err != nil {
		return nil
	}
	if err := u.UnmarshalJSON(buf.Bytes()); err != nil {
		return nil
	}
	return u
}

const crdKeyPrefix = "/kubernetes.io/apiextensions.k8s.io/customresourcedefinitions/"

// buildEtcdKeyIndex scans the snapshot once and records every kube key together
// with its etcd storage key in koff.KubeKeysToEtcdKeys. It only decodes the
// protobuf key field (not the potentially large value), and also verifies the
// snapshot integrity.
func buildEtcdKeyIndex(koff *types.KoffCommand) error {
	return koff.EtcdDb.View(func(tx *bolt.Tx) error {
		// check snapshot file integrity first
		var dbErrStrings []string
		for dbErr := range tx.Check() {
			dbErrStrings = append(dbErrStrings, dbErr.Error())
		}
		if len(dbErrStrings) > 0 {
			return fmt.Errorf("snapshot file integrity check failed. %d errors found.\n"+strings.Join(dbErrStrings, "\n"), len(dbErrStrings))
		}
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("bucket %q not found", "key")
		}
		return b.ForEach(func(k, value []byte) error {
			// Fast path: extract only the kube key (protobuf field 1) without
			// decoding the whole value. The value can be large (secrets,
			// configmaps, ...) and is only needed when the record is rendered.
			key, ok := protoKeyField(value)
			if !ok {
				var kv mvccpb.KeyValue
				if err := kv.Unmarshal(value); err != nil {
					return fmt.Errorf("unmarshalling etcd value: %w", err)
				}
				key = kv.Key
			}
			koff.KubeKeysToEtcdKeys[string(key)] = k
			return nil
		})
	})
}

// loadCRDsFromEtcd decodes the CustomResourceDefinition records already indexed
// in koff.KubeKeysToEtcdKeys and registers their aliases. It is only invoked
// when a request actually references a custom resource.
func loadCRDsFromEtcd(koff *types.KoffCommand) error {
	return koff.EtcdDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("bucket %q not found", "key")
		}
		for kubeKey, etcdKey := range koff.KubeKeysToEtcdKeys {
			if !strings.HasPrefix(kubeKey, crdKeyPrefix) {
				continue
			}
			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(b.Get(etcdKey)); err != nil {
				return fmt.Errorf("unmarshalling CRD value: %w", err)
			}
			registerCRDAliases(koff, kubeKey, kv.Value)
		}
		return nil
	})
}

// etcdRequestNeedsCRDs reports whether any requested resource type is not a
// built-in known resource and therefore requires CRD definitions to resolve.
func etcdRequestNeedsCRDs(koff *types.KoffCommand, args []string) bool {
	for _, resourceType := range requestedResourceTypes(args) {
		if _, ok := koff.KnownResources[resourceType]; !ok {
			return true
		}
	}
	return false
}

// requestedResourceTypes extracts the resource-type tokens from get arguments,
// mirroring how ParseGetArgs interprets them (comma lists, resource/name form,
// and "type name..." form), with any group suffix stripped.
func requestedResourceTypes(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	lower := make([]string, len(args))
	hasSlash := false
	for i, a := range args {
		lower[i] = strings.ToLower(a)
		if strings.Contains(lower[i], "/") {
			hasSlash = true
		}
	}

	var rawTypes []string
	switch {
	case len(lower) == 1 && !strings.Contains(lower[0], "/"):
		rawTypes = strings.Split(strings.Trim(lower[0], ","), ",")
	case hasSlash:
		for _, a := range lower {
			rawTypes = append(rawTypes, strings.SplitN(a, "/", 2)[0])
		}
	default: // "type name..." form: only the first arg is a type
		rawTypes = append(rawTypes, lower[0])
	}

	types := make([]string, 0, len(rawTypes))
	for _, t := range rawTypes {
		if t == "" {
			continue
		}
		if i := strings.Index(t, "."); i >= 0 {
			t = t[:i]
		}
		types = append(types, t)
	}
	return types
}

// registerCRDAliases parses a CustomResourceDefinition value and registers every
// way it can be referenced (kind, plural, singular, short names, singular.group)
// into koff.EtcdAliasToCrdKubeKey.
func registerCRDAliases(koff *types.KoffCommand, kubeKey string, crdValue []byte) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(crdValue, crd); err != nil {
		return
	}
	sub := types.AliasSubField{
		Kind:       strings.ToLower(crd.Spec.Names.Kind),
		Plural:     strings.ToLower(crd.Spec.Names.Plural),
		KubeKey:    kubeKey,
		Group:      strings.ToLower(crd.Spec.Group),
		Namespaced: string(crd.Spec.Scope) == "Namespaced",
	}
	aliases := []string{
		strings.ToLower(crd.Spec.Names.Kind),
		strings.ToLower(crd.Spec.Names.Plural),
		strings.ToLower(crd.Spec.Names.Singular),
		crd.Spec.Names.Singular + "." + crd.Spec.Group,
	}
	aliases = append(aliases, crd.Spec.Names.ShortNames...)
	for _, alias := range aliases {
		if alias == "" || alias == "." {
			continue
		}
		koff.EtcdAliasToCrdKubeKey[strings.ToLower(alias)] = sub
	}
}

// protoKeyField extracts the bytes of protobuf field 1 (wire type 2) from a
// serialized mvccpb.KeyValue, which is always the record's key. It returns
// false if the buffer does not start with that field so the caller can fall
// back to a full unmarshal.
func protoKeyField(value []byte) ([]byte, bool) {
	// Tag for field 1, wire type 2 (length-delimited) is 0x0a.
	if len(value) == 0 || value[0] != 0x0a {
		return nil, false
	}
	length, n := binary.Uvarint(value[1:])
	if n <= 0 {
		return nil, false
	}
	start := 1 + n
	end := start + int(length)
	if end < start || end > len(value) {
		return nil, false
	}
	return value[start:end], true
}

// GetResourcesFromEtcd selects, among all kube keys already discovered in
// koff.KubeKeysToEtcdKeys, those matching one of the requested prefixes and
// appends them to koff.EtcdKubeKeysToGet. It operates on the in-memory key set
// built by populateCRDsFromEtcd, avoiding a second full scan of the snapshot.
func GetResourcesFromEtcd(koff *types.KoffCommand, etcdKeyPrefixesToCheck map[string]bool) error {
	for kubeKey := range koff.KubeKeysToEtcdKeys {
		if matchesPrefix(koff, kubeKey, etcdKeyPrefixesToCheck) {
			koff.EtcdKubeKeysToGet = append(koff.EtcdKubeKeysToGet, kubeKey)
		}
	}
	return nil
}

// matchesPrefix reports whether a kube key belongs to one of the requested
// resource prefixes, accounting for cluster-scoped vs namespaced layouts and
// the --all-namespaces flag.
func matchesPrefix(koff *types.KoffCommand, kubeKey string, prefixes map[string]bool) bool {
	if _, ok := prefixes[kubeKey]; ok {
		return true
	}
	parts := strings.Split(kubeKey, "/")
	switch len(parts) {
	case 4:
		// cluster-scoped resource, e.g. /kubernetes.io/namespaces/<name>
		if _, ok := prefixes[strings.Join(parts[:3], "/")]; ok {
			return true
		}
	case 5:
		// namespaced resource, e.g. /kubernetes.io/pods/<ns>/<name>
		if koff.AllNamespaces && !strings.Contains(parts[2], ".") {
			if _, ok := prefixes[strings.Join(parts[:3], "/")]; ok {
				return true
			}
		} else if koff.AllNamespaces && strings.Contains(parts[2], ".") {
			if _, ok := prefixes[strings.Join(parts[:4], "/")]; ok {
				return true
			}
		} else {
			if _, ok := prefixes[strings.Join(parts[:4], "/")]; ok {
				return true
			}
		}
	case 6:
		// namespaced resource with sub-path, e.g.
		// /kubernetes.io/services/specs/<ns>/<name>
		if koff.AllNamespaces {
			if _, ok := prefixes[strings.Join(parts[:4], "/")]; ok {
				return true
			}
		} else {
			if _, ok := prefixes[strings.Join(parts[:5], "/")]; ok {
				return true
			}
		}
	}
	return false
}
