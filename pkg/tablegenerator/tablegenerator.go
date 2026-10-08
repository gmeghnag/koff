package tablegenerator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"

	//"github.com/gmeghnag/koff/types"
	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func InternalResourceTable(Koff *types.KoffCommand, runtimeObject runtime.Object, unstruct *unstructured.Unstructured) (*metav1.Table, error) {
	resourceKind := strings.ToLower(unstruct.GetKind())
	table, err := Koff.TableGenerator.GenerateTable(runtimeObject, printers.GenerateOptions{Wide: Koff.Wide, NoHeaders: false})
	if err != nil {
		return table, err
	}
	// Defensive guard: every mutation below assumes at least one row and one
	// column. Some handlers can legitimately emit an empty table; bail out
	// early instead of panicking with an index-out-of-range.
	if len(table.Rows) == 0 || len(table.ColumnDefinitions) == 0 {
		return table, err
	}
	for i, column := range table.ColumnDefinitions {
		if column.Name == "Age" {
			table.Rows[0].Cells[i] = helpers.TranslateTimestamp(unstruct.GetCreationTimestamp())
			if unstruct.GetKind() != "Node" {
				break
			}
		}
		if column.Name == "Roles" {
			var NodeRoles []string
			for i := range unstruct.GetLabels() {
				if strings.HasPrefix(i, "node-role.kubernetes.io/") {
					NodeRoles = append(NodeRoles, strings.Split(i, "/")[1])
				}
			}
			sort.Strings(NodeRoles)
			if len(NodeRoles) > 0 {
				table.Rows[0].Cells[i] = strings.Join(NodeRoles, ",")
			}

		}
	}
	if table.ColumnDefinitions[0].Name == "Name" {
		if Koff.ShowKind || Koff.Namespace == "" || len(Koff.GetArgs) != 1 {
			if unstruct.GetAPIVersion() == "v1" {
				table.Rows[0].Cells[0] = resourceKind + "/" + unstruct.GetName()
			} else {
				table.Rows[0].Cells[0] = resourceKind + "." + unstruct.GetObjectKind().GroupVersionKind().Group + "/" + unstruct.GetName()
			}
		} else {
			table.Rows[0].Cells[0] = unstruct.GetName()
		}
	}

	if (Koff.ShowNamespace || Koff.AllNamespaces) && unstruct.GetNamespace() != "" {
		table.ColumnDefinitions = append([]metav1.TableColumnDefinition{{Format: "string", Name: "Namespace"}}, table.ColumnDefinitions...)
		table.Rows[0].Cells = append([]any{unstruct.GetNamespace()}, table.Rows[0].Cells...)
	}
	return table, err
}

// GenerateCustomResourceTable renders a custom resource, resolving its CRD
// (caching it in Koff.CRD for runs of the same kind). It is used by the serial
// path; the parallel path uses ResolveCRDReadOnly + BuildCustomResourceTable.
func GenerateCustomResourceTable(Koff *types.KoffCommand, unstruct unstructured.Unstructured) (*metav1.Table, error) {
	// Resolve the CRD only when the object kind differs from the previous one.
	if Koff.CurrentKind != unstruct.GetKind() {
		Koff.CRD = resolveCRD(Koff, unstruct)
	}
	return BuildCustomResourceTable(Koff, unstruct, Koff.CRD)
}

// resolveCRD finds the CRD for a custom resource, consulting the in-memory
// caches and, for file mode, scanning ~/.koff once. It mutates Koff caches and
// must be called serially.
func resolveCRD(Koff *types.KoffCommand, unstruct unstructured.Unstructured) *apiextensionsv1.CustomResourceDefinition {
	resourceKind := strings.ToLower(unstruct.GetKind())
	if crd, ok := Koff.AliasToCrd[resourceKind]; ok {
		return &apiextensionsv1.CustomResourceDefinition{Spec: crd.Spec}
	}
	if Koff.IsEtcdDb {
		crd, err := GetCrdFromCr(Koff, resourceKind+"."+unstruct.GetObjectKind().GroupVersionKind().Group)
		if err != nil {
			return nil
		}
		return crd
	}
	helpers.RetrieveKindGroupFromCRDS(Koff, resourceKind)
	if crd, ok := Koff.AliasToCrd[resourceKind]; ok {
		return &apiextensionsv1.CustomResourceDefinition{Spec: crd.Spec}
	}
	return nil
}

// ResolveCRDReadOnly resolves a CRD without mutating Koff, so it is safe for
// concurrent use. It consults the already-loaded alias caches (etcd preloads
// them before rendering) and, for etcd, reads the CRD via a read-only
// transaction; it never triggers a ~/.koff disk scan.
func ResolveCRDReadOnly(Koff *types.KoffCommand, unstruct unstructured.Unstructured) *apiextensionsv1.CustomResourceDefinition {
	resourceKind := strings.ToLower(unstruct.GetKind())
	if crd, ok := Koff.AliasToCrd[resourceKind]; ok {
		return &apiextensionsv1.CustomResourceDefinition{Spec: crd.Spec}
	}
	if Koff.IsEtcdDb {
		crd, err := GetCrdFromCr(Koff, resourceKind+"."+unstruct.GetObjectKind().GroupVersionKind().Group)
		if err != nil {
			return nil
		}
		return crd
	}
	return nil
}

// BuildCustomResourceTable builds the table for a custom resource given its
// (possibly nil) CRD. It only reads Koff configuration and is safe for
// concurrent use.
func BuildCustomResourceTable(Koff *types.KoffCommand, unstruct unstructured.Unstructured, crd *apiextensionsv1.CustomResourceDefinition) (*metav1.Table, error) {
	resourceKind := strings.ToLower(unstruct.GetKind())
	table := &metav1.Table{}
	if crd == nil {
		//fmt.Println("CustomResourceDefinition not found for kind \"" + unstruct.GetKind() + "\", apiVersion: \"" + unstruct.GetAPIVersion() + "\"")
		//return table, fmt.Errorf("CustomResourceDefinition not found for kind \"" + unstruct.GetKind() + "\", apiVersion: \"" + unstruct.GetAPIVersion() + "\"")
		if (Koff.ShowNamespace || Koff.AllNamespaces) && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{
				{Name: "Namespace", Type: "string", Format: "name"},
				{Name: "Name", Type: "string", Format: "string"},
				{Name: "Created At", Type: "date"},
			}
			if Koff.ShowKind || Koff.Namespace == "" || len(Koff.GetArgs) != 1 {
				table.Rows = []metav1.TableRow{{Cells: []any{unstruct.GetNamespace(), resourceKind + "." + unstruct.GetObjectKind().GroupVersionKind().Group + "/" + unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			} else {
				table.Rows = []metav1.TableRow{{Cells: []any{unstruct.GetNamespace(), unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			}

		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{
				{Name: "Name", Type: "string", Format: "name"},
				{Name: "Created At", Type: "date"},
			}
			if Koff.ShowKind || Koff.Namespace == "" || len(Koff.GetArgs) != 1 {
				table.Rows = []metav1.TableRow{{Cells: []any{resourceKind + "." + unstruct.GetObjectKind().GroupVersionKind().Group + "/" + unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}

			} else {
				table.Rows = []metav1.TableRow{{Cells: []any{unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			}

		}
		return table, nil
	}

	cells := []any{}
	// table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
	if Koff.ShowKind || Koff.Namespace == "" || len(Koff.GetArgs) != 1 {
		if (Koff.ShowNamespace || Koff.AllNamespaces) && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Namespace", Format: "string"}, {Name: "Name", Format: "name"}}
			cells = []any{unstruct.GetNamespace(), resourceKind + "." + unstruct.GetObjectKind().GroupVersionKind().Group + "/" + unstruct.GetName()}
		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
			cells = []any{resourceKind + "." + unstruct.GetObjectKind().GroupVersionKind().Group + "/" + unstruct.GetName()}
		}
	} else {
		if (Koff.ShowNamespace || Koff.AllNamespaces) && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Namespace", Format: "string"}, {Name: "Name", Format: "name"}}
			cells = []any{unstruct.GetNamespace(), unstruct.GetName()}
		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
			cells = []any{unstruct.GetName()}
		}
	}
	if len(crd.Spec.AdditionalPrinterColumns) > 0 {
		for _, column := range crd.Spec.AdditionalPrinterColumns {
			table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: column.Name, Format: "string"})
			if column.Name == "Age" {
				cells = append(cells, helpers.TranslateTimestamp(unstruct.GetCreationTimestamp()))
			}
			if column.Name == "Since" {
				v := helpers.GetFromJsonPath(unstruct.Object, fmt.Sprintf("%s%s%s", "{", column.JSONPath, "}"))
				parsedTime, _ := time.Parse(time.RFC3339, v)
				metav1Time := metav1.Time{Time: parsedTime}
				v = helpers.TranslateTimestamp(metav1Time)
				cells = append(cells, v)
			} else {
				v := helpers.GetFromJsonPath(unstruct.Object, fmt.Sprintf("%s%s%s", "{", column.JSONPath, "}"))
				cells = append(cells, v)
			}
		}
	} else {
		for i, column := range crd.Spec.Versions {
			if (crd.Spec.Group + "/" + column.Name) == unstruct.GetAPIVersion() {
				if len(crd.Spec.Versions[i].AdditionalPrinterColumns) > 0 {
					for _, column := range crd.Spec.Versions[i].AdditionalPrinterColumns {
						table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: column.Name, Format: "string"})
						if column.Name == "Age" {
							cells = append(cells, helpers.TranslateTimestamp(unstruct.GetCreationTimestamp()))
						}
						if column.Name == "Since" {
							v := helpers.GetFromJsonPath(unstruct.Object, fmt.Sprintf("%s%s%s", "{", column.JSONPath, "}"))
							parsedTime, _ := time.Parse(time.RFC3339, v)
							metav1Time := metav1.Time{Time: parsedTime}
							v = helpers.TranslateTimestamp(metav1Time)
							cells = append(cells, v)
						} else {
							v := helpers.GetFromJsonPath(unstruct.Object, fmt.Sprintf("%s%s%s", "{", column.JSONPath, "}"))
							cells = append(cells, v)
						}
					}
				} else {
					table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: "Age", Format: "string"})
					cells = append(cells, helpers.TranslateTimestamp(unstruct.GetCreationTimestamp()))
				}
				break
			}
		}
	}
	table.Rows = []metav1.TableRow{{Cells: cells}}

	return table, nil
}

func GetCrdFromCr(Koff *types.KoffCommand, cr string) (*apiextensionsv1.CustomResourceDefinition, error) {
	crFields := Koff.EtcdAliasToCrdKubeKey[cr]
	crKubeKey := "/kubernetes.io/apiextensions.k8s.io/customresourcedefinitions/" + crFields.Plural + "." + crFields.Group
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := Koff.EtcdDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("bucket %q not found", "key")
		}
		etcdKey, ok := Koff.KubeKeysToEtcdKeys[crKubeKey]
		if !ok {
			return fmt.Errorf("CRD %q not found in snapshot", crKubeKey)
		}
		var kv mvccpb.KeyValue
		if err := kv.Unmarshal(b.Get(etcdKey)); err != nil {
			return fmt.Errorf("unmarshalling CRD value: %w", err)
		}
		return yaml.Unmarshal(kv.Value, crd)
	})
	if err != nil {
		return crd, err
	}
	return crd, nil
}
