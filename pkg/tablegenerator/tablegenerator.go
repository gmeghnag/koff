package tablegenerator

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/printers"
	"sigs.k8s.io/yaml"

	//"github.com/gmeghnag/koff/types"

	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func InternalResourceTable(Koff *types.KoffCommand, runtimeObject runtime.Object, unstruct *unstructured.Unstructured) (*metav1.Table, error) {
	table, err := Koff.TableGenerator.GenerateTable(runtimeObject, printers.GenerateOptions{Wide: Koff.Wide, NoHeaders: false})
	if err != nil {
		return table, err
	}
	for i, column := range table.ColumnDefinitions {
		if column.Name == "Age" {
			table.Rows[0].Cells[i] = helpers.TranslateTimestamp(unstruct.GetCreationTimestamp())
			break
		}
	}
	if table.ColumnDefinitions[0].Name == "Name" {
		if Koff.ShowKind == true {
			if unstruct.GetAPIVersion() == "v1" {
				table.Rows[0].Cells[0] = strings.ToLower(unstruct.GetKind()) + "/" + unstruct.GetName()
			} else {
				table.Rows[0].Cells[0] = strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()
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

func UndefinedResourceTable(Koff *types.KoffCommand, unstruct unstructured.Unstructured) *metav1.Table {
	table := &metav1.Table{}
	if Koff.ShowNamespace == true && unstruct.GetNamespace() != "" {
		table.ColumnDefinitions = []metav1.TableColumnDefinition{
			{Name: "Namespace", Type: "string", Format: "name"},
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Created At", Type: "date"}, // Priority: 1
		}
		table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetNamespace(), unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
	} else {
		table.ColumnDefinitions = []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Created At", Type: "date"}, // Priority: 1
		}
		table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
	}
	return table
}

func GenerateCustomResourceTable(Koff *types.KoffCommand, unstruct unstructured.Unstructured) (*metav1.Table, error) {
	table := &metav1.Table{}
	// search for its corresponding CRD obly if this object Kind differs from the previous one parsed
	if Koff.CurrentKind != unstruct.GetObjectKind().GroupVersionKind().Kind {
		Koff.CRD = nil
		home, _ := os.UserHomeDir()
		crdsPath := home + "/.koff/customresourcedefinitions/"
		objectKind := unstruct.GetKind()
		if strings.HasSuffix(objectKind, ".config") {
			objectKind = strings.Replace(objectKind, ".config", ".config.openshift.io", -1)
		}
		_, err := helpers.Exists(crdsPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		crds, _ := ioutil.ReadDir(crdsPath)
		for _, f := range crds {
			crdYamlPath := crdsPath + f.Name()
			crdByte, _ := ioutil.ReadFile(crdYamlPath)
			_crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := yaml.Unmarshal([]byte(crdByte), &_crd); err != nil {
				fmt.Fprintln(os.Stderr, "Error when trying to unmarshal file", crdYamlPath)
				os.Exit(1)
			}
			if strings.ToLower(_crd.Spec.Names.Kind) == strings.ToLower(objectKind) {
				Koff.CRD = _crd
				klog.V(4).Info("INFO ", fmt.Sprintf("\"%s\" found in path \"%s\"", unstruct.GetKind(), crdYamlPath))
				break
			}
			klog.V(4).Info("INFO ", fmt.Sprintf("\"%s\" not found in path \"%s\"", unstruct.GetKind(), crdYamlPath))
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
			if Koff.ShowKind == true {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetNamespace(), strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			} else {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetNamespace(), unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			}

		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{
				{Name: "Name", Type: "string", Format: "name"},
				{Name: "Created At", Type: "date"},
			}
			if Koff.ShowKind == true {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}

			} else {
				table.Rows = []metav1.TableRow{{Cells: []interface{}{unstruct.GetName(), unstruct.GetCreationTimestamp().Time.UTC().Format("2006-01-02T15:04:05")}}}
			}

		}
		return table, nil
	}

	cells := []interface{}{}
	// table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
	if Koff.ShowKind == true {
		if Koff.ShowNamespace && unstruct.GetNamespace() != "" {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Namespace", Format: "string"}, {Name: "Name", Format: "name"}}
			cells = []interface{}{unstruct.GetNamespace(), strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()}
		} else {
			table.ColumnDefinitions = []metav1.TableColumnDefinition{{Name: "Name", Format: "name"}}
			cells = []interface{}{strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()}
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
