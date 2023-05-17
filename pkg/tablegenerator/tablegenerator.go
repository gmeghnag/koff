package tablegenerator

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"

	//"github.com/gmeghnag/koff/types"

	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func InternalResourceTable(Koff *types.KoffCommand, runtimeObject runtime.Object, unstruct *unstructured.Unstructured) (*metav1.Table, error) {
	resourceKind := strings.ToLower(unstruct.GetKind())
	table, err := Koff.TableGenerator.GenerateTable(runtimeObject, printers.GenerateOptions{Wide: Koff.Wide, NoHeaders: false})
	if err != nil {
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
		if Koff.ShowKind == true || Koff.Namespace == "" {
			if unstruct.GetAPIVersion() == "v1" {
				table.Rows[0].Cells[0] = resourceKind + "/" + unstruct.GetName()
			} else {
				table.Rows[0].Cells[0] = resourceKind + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()
			}
		} else {
			table.Rows[0].Cells[0] = unstruct.GetName()
		}
	}

	if Koff.ShowNamespace && unstruct.GetNamespace() != "" {
		table.ColumnDefinitions = append([]metav1.TableColumnDefinition{{Format: "string", Name: "Namespace"}}, table.ColumnDefinitions...)
		table.Rows[0].Cells = append([]interface{}{unstruct.GetNamespace()}, table.Rows[0].Cells...)
	}
	return table, err
}

func GenerateCustomResourceTable(Koff *types.KoffCommand, unstruct unstructured.Unstructured) (*metav1.Table, error) {
	resourceKind := strings.ToLower(unstruct.GetKind())
	table := &metav1.Table{}
	// search for its corresponding CRD obly if this object Kind differs from the previous one parsed
	if Koff.CurrentKind != unstruct.GetKind() {
		Koff.CRD = nil
		crd, ok := Koff.AliasToCrd[resourceKind]
		if ok {
			_crd := &apiextensionsv1.CustomResourceDefinition{Spec: crd.Spec}
			Koff.CRD = _crd
		} else {
			helpers.RetrieveKindGroupFromCRDS(Koff, resourceKind)
			crd, ok := Koff.AliasToCrd[resourceKind]
			if ok {
				_crd := &apiextensionsv1.CustomResourceDefinition{Spec: crd.Spec}
				Koff.CRD = _crd
			}
		}
	}
	if Koff.CRD == nil {
		//fmt.Println("CustomResourceDefinition not found for kind \"" + unstruct.GetKind() + "\", apiVersion: \"" + unstruct.GetAPIVersion() + "\"")
		//return table, fmt.Errorf("CustomResourceDefinition not found for kind \"" + unstruct.GetKind() + "\", apiVersion: \"" + unstruct.GetAPIVersion() + "\"")
		if Koff.ShowNamespace == true && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{
				{Name: "Namespace", Type: "string", Format: "name"},
				{Name: "Name", Type: "string", Format: "string"},
				{Name: "Created At", Type: "date"},
			}
			if Koff.ShowKind == true || Koff.Namespace == "" {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetNamespace(), resourceKind + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			} else {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetNamespace(), unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			}

		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{
				{Name: "Name", Type: "string", Format: "name"},
				{Name: "Created At", Type: "date"},
			}
			if Koff.ShowKind == true || Koff.Namespace == "" {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{resourceKind + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}

			} else {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			}

		}
		return table, nil
	}

	cells := []interface{}{}
	// table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
	if Koff.ShowKind == true || Koff.Namespace == "" {
		if Koff.ShowNamespace && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Namespace", Format: "string"}, {Name: "Name", Format: "name"}}
			cells = []interface{}{unstruct.GetNamespace(), resourceKind + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()}
		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
			cells = []interface{}{resourceKind + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()}
		}
	} else {
		if Koff.ShowNamespace && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Namespace", Format: "string"}, {Name: "Name", Format: "name"}}
			cells = []interface{}{unstruct.GetNamespace(), unstruct.GetName()}
		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
			cells = []interface{}{unstruct.GetName()}
		}
	}
	for i, column := range Koff.CRD.Spec.Versions {
		if (Koff.CRD.Spec.Group + "/" + column.Name) == unstruct.GetAPIVersion() {
			if len(Koff.CRD.Spec.Versions[i].AdditionalPrinterColumns) > 0 {
				for _, column := range Koff.CRD.Spec.Versions[i].AdditionalPrinterColumns {
					table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: column.Name, Format: "string"})
					if column.Name == "Age" || column.Type == "date" {
						cells = append(cells, helpers.TranslateTimestamp(unstruct.GetCreationTimestamp()))
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
	table.Rows = []metav1.TableRow{{Cells: cells}}

	return table, nil
}
