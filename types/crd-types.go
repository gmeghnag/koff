package types

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"bytes"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/yaml"
)

var data [][]string
var headers []string

func (Koff *KoffCommand) GenerateCustomResourceTable(unstruct unstructured.Unstructured) (*metav1.Table, error) {
	table := &metav1.Table{}
	if Koff.CurrentKind != unstruct.GetObjectKind().GroupVersionKind().Kind {
		Koff.CRD = nil
		home, _ := os.UserHomeDir()
		crdsPath := home + "/.koff/customresourcedefinitions/"
		objectKind := unstruct.GetKind()
		if strings.HasSuffix(objectKind, ".config") {
			objectKind = strings.Replace(objectKind, ".config", ".config.openshift.io", -1)
		}
		_, err := Exists(crdsPath)
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
					cells = append(cells, translateTimestamp(unstruct.GetCreationTimestamp()))
				} else {
					v := getFromJsonPath(unstruct.Object, fmt.Sprintf("%s%s%s", "{", column.JSONPath, "}"))
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

func getFromJsonPath(data interface{}, jsonPathTemplate string) string {
	buf := new(bytes.Buffer)
	jPath := jsonpath.New("out")
	jPath.AllowMissingKeys(false)
	jPath.EnableJSONOutput(false)
	err := jPath.Parse(jsonPathTemplate)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: error parsing jsonpath "+jsonPathTemplate+", "+err.Error())
		os.Exit(1)
	}
	jPath.Execute(buf, data)
	return buf.String()
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
