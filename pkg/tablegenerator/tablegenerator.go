package tablegenerator

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"
	"sigs.k8s.io/yaml"

	//"github.com/gmeghnag/koff/types"

	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func InternalResourceTable(Koff *types.KoffCommand, runtimeObject runtime.Object, unstruct *unstructured.Unstructured) (*metav1.Table, error) {
	table, err := Koff.TableGenerator.GenerateTable(runtimeObject, printers.GenerateOptions{Wide: false})
	if err != nil {
		return table, err
	}
	//if table.ColumnDefinitions[0].Name == "Name" {
	//	table.Rows[0].Cells[0] = unstruct.GetName()
	//}
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
				break
			}
		}
	}
	if Koff.CRD == nil {
		//fmt.Println("CustomResourceDefinition not found for kind \"" + unstruct.GetKind() + "\", apiVersion: \"" + unstruct.GetAPIVersion() + "\"")
		return table, fmt.Errorf("CustomResourceDefinition not found for kind \"" + unstruct.GetKind() + "\", apiVersion: \"" + unstruct.GetAPIVersion() + "\"")
		//table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: "Name", Format: "name"})
		//cells := []interface{}{unstruct.GetKind()}
		//newRow := metav1.TableRow{
		//	Cells: cells,
		//}
		//table.Rows = []metav1.TableRow{newRow}
		//return table, nil
	}

	table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: "Name", Format: "name"})
	cells := []interface{}{""}
	for i, column := range Koff.CRD.Spec.Versions {
		if (Koff.CRD.Spec.Group + "/" + column.Name) == unstruct.GetAPIVersion() {
			for _, column := range Koff.CRD.Spec.Versions[i].AdditionalPrinterColumns {
				table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{Name: column.Name, Format: "name"})
				if column.Name == "Age" || column.Type == "date" {
					cells = append(cells, helpers.TranslateTimestamp(unstruct.GetCreationTimestamp()))
				} else {
					v := helpers.GetFromJsonPath(unstruct.Object, fmt.Sprintf("%s%s%s", "{", column.JSONPath, "}"))
					cells = append(cells, v)
				}
			}
			break
		}
	}
	newRow := metav1.TableRow{
		Cells: cells,
	}
	table.Rows = []metav1.TableRow{newRow}

	return table, nil
}
