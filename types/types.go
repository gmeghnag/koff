package types

// specific labels https://github.com/seans3/kubernetes/blob/6108dac6708c026b172f3928e137c206437791da/pkg/printers/internalversion/printers_test.go#L1979
import (
	"bytes"
	"log"
	"os"
	"strings"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	// "k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/apimachinery/pkg/api/meta"
	//runtime "k8s.io/apimachinery/pkg/runtime"
	//utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	appsv1printer "github.com/openshift/openshift-apiserver/pkg/apps/printers/internalversion"
	authorizationprinters "github.com/openshift/openshift-apiserver/pkg/authorization/printers/internalversion"
	buildprinters "github.com/openshift/openshift-apiserver/pkg/build/printers/internalversion"
	imageprinters "github.com/openshift/openshift-apiserver/pkg/image/printers/internalversion"

	//core "k8s.io/kubernetes/pkg/apis/core"
	//ocpinternal "github.com/openshift/openshift-apiserver/pkg/apps/printers/internalversion"

	"k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"

	// cliprint "k8s.io/cli-runtime/pkg/printers"
	_ "embed"

	runtime "k8s.io/apimachinery/pkg/runtime"
	cliprint "k8s.io/cli-runtime/pkg/printers"
)

var RuntimeObjectType runtime.Object

var dataIn []byte
var Koff = NewKoffCommand()

func (Koff *KoffCommand) rawObjectToTable(rawObject []byte, unstructuredObject unstructured.Unstructured) *metav1.Table {
	// needs code review
	unstruct := &unstructuredObject
	RuntimeObjectType := rawObjectToRuntimeObject(rawObject, Koff.Schema)
	if err := yaml.Unmarshal([]byte(rawObject), RuntimeObjectType); err != nil {
		//log.Printf(".... Error: %s\n", err)
	}
	table, err := internalResourceTable(RuntimeObjectType, unstruct)
	if err != nil {
		// printer for the object is not registered or is a crd
		//log.printf fmt.Println(err, unstruct.GetKind(), unstruct.GetAPIVersion())
		table, err = Koff.GenerateCustomResourceTable(*unstruct)
		if err != nil {
			table = undefinedResourceTable(*unstruct)

		}

	}

	// TODO Move it into the specific TableConverter method/function
	// to prevent looping again over the ColumnDefinitions
	//if Koff.ShowKind == true {
	//	if unstruct.GetAPIVersion() == "v1" {
	//		table.Rows[0].Cells[0] = strings.ToLower(unstruct.GetKind()) + "/" + unstruct.GetName()
	//	} else {
	//		table.Rows[0].Cells[0] = strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()
	//	}
	//} else {
	//	table.Rows[0].Cells[0] = unstruct.GetName()
	//}
	//if unstruct.GetNamespace() != "" {
	//	namespaceRaw := metav1.TableRow{}
	//	namespaceRaw.Cells = append(namespaceRaw.Cells, unstruct.GetNamespace())
	//	namespaceRaw.Cells = append(namespaceRaw.Cells, table.Rows[0].Cells...)
	//	table.Rows[0] = namespaceRaw
	//	columnWithNamespace := []metav1.TableColumnDefinition{}
	//	columnWithNamespace = append(columnWithNamespace, metav1.TableColumnDefinition{Name: "NAMESPACE"})
	//	columnWithNamespace = append(columnWithNamespace, table.ColumnDefinitions...)
	//	table.ColumnDefinitions = columnWithNamespace
	//}

	return table

}

type KoffCommand struct {
	Kind             string
	Namespace        string
	Wide             bool
	ShowLabels       bool
	SingleResource   bool
	Items            []unstructured.UnstructuredList
	Test             bytes.Buffer
	LastResourceType runtime.Object
	CurrentKind      string
	LastObj          unstructured.Unstructured
	Schema           *runtime.Scheme
	Printer          cliprint.ResourcePrinter
	Table            metav1.Table
	TableGenerator   *printers.HumanReadableGenerator
	CRD              *apiextensionsv1.CustomResourceDefinition
	FromInput        bool
	ShowKind         bool
	ShowNamespace    bool
}

func (Koff *KoffCommand) HandleObject(obj unstructured.Unstructured) {
	Koff.LastObj = obj
	rawObject, err := yaml.Marshal(obj.Object)
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// INIZIO
	// objectTable :=Koff.rawObjectToTable(rawObject, obj)
	RuntimeObjectType := rawObjectToRuntimeObject(rawObject, Koff.Schema)
	if err := yaml.Unmarshal([]byte(rawObject), RuntimeObjectType); err != nil {
		//log.Printf(".... Error: %s\n", err)
	}
	objectTable, err := internalResourceTable(RuntimeObjectType, &obj)
	if err != nil {
		// printer for the object is not registered or is a crd
		//log.printf fmt.Println(err, unstruct.GetKind(), unstruct.GetAPIVersion())
		objectTable, err = Koff.GenerateCustomResourceTable(obj)
		if err != nil {
			objectTable = undefinedResourceTable(obj)

		}

	}
	// END
	// se l'oggetto è uguale a quello precedente
	// non printo newTable e non aggiungo ColumnDefinitions
	if Koff.CurrentKind == obj.GetObjectKind().GroupVersionKind().Kind {
		Koff.Table.Rows = append(Koff.Table.Rows, objectTable.Rows...)
	} else {
		// printo la tabella dell'oggetto precedente
		printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: false, Wide: false, WithNamespace: false})
		err = printer.PrintObj(&Koff.Table, &Koff.Test)
		if err != nil {
			log.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		if Koff.CurrentKind != "" {
			Koff.Test.WriteByte('\n')
		}
		Koff.CurrentKind = obj.GetObjectKind().GroupVersionKind().Kind
		Koff.Table = metav1.Table{}
		Koff.Table.ColumnDefinitions = append(Koff.Table.ColumnDefinitions, objectTable.ColumnDefinitions...)
		Koff.Table.Rows = append(Koff.Table.Rows, objectTable.Rows...)
	}
}

func (Koff *KoffCommand) InitializeTableGenerator() {
	Koff.TableGenerator = printers.NewTableGenerator()
	AddKoffHandlers(Koff.TableGenerator)
	printersinternal.AddHandlers(Koff.TableGenerator)
	buildprinters.AddBuildOpenShiftHandlers(Koff.TableGenerator)
	appsv1printer.AddAppsOpenShiftHandlers(Koff.TableGenerator)
	authorizationprinters.AddAuthorizationOpenShiftHandler(Koff.TableGenerator)
	imageprinters.AddImageOpenShiftHandlers(Koff.TableGenerator)
}

func undefinedResourceTable(unstruct unstructured.Unstructured) *metav1.Table {
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

func internalResourceTable(runtimeObject runtime.Object, unstruct *unstructured.Unstructured) (*metav1.Table, error) {
	table, err := Koff.TableGenerator.GenerateTable(runtimeObject, printers.GenerateOptions{Wide: false})
	if err != nil {
		return table, err
	}
	//if table.ColumnDefinitions[0].Name == "Name" {
	//	table.Rows[0].Cells[0] = unstruct.GetName()
	//}
	for i, column := range table.ColumnDefinitions {
		if column.Name == "Age" {
			table.Rows[0].Cells[i] = translateTimestamp(unstruct.GetCreationTimestamp())
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
