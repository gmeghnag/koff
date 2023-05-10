package types

// specific labels https://github.com/seans3/kubernetes/blob/6108dac6708c026b172f3928e137c206437791da/pkg/printers/internalversion/printers_test.go#L1979
import (
	"bytes"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

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

//	func (Koff *KoffCommand) rawObjectToTable(rawObject []byte, unstructuredObject unstructured.Unstructured) *metav1.Table {
//		// needs code review
//		unstruct := &unstructuredObject
//		RuntimeObjectType := rawObjectToRuntimeObject(rawObject, Koff.Schema)
//		if err := yaml.Unmarshal([]byte(rawObject), RuntimeObjectType); err != nil {
//			//log.Printf(".... Error: %s\n", err)
//		}
//		table, err := tablegenerator.InternalResourceTable(RuntimeObjectType, unstruct)
//		if err != nil {
//			// printer for the object is not registered or is a crd
//			//log.printf fmt.Println(err, unstruct.GetKind(), unstruct.GetAPIVersion())
//			table, err = Koff.GenerateCustomResourceTable(*unstruct)
//			if err != nil {
//				table = undefinedResourceTable(*unstruct)
//
//			}
//
//		}
//
//		// TODO Move it into the specific TableConverter method/function
//		// to prevent looping again over the ColumnDefinitions
//		//if Koff.ShowKind == true {
//		//	if unstruct.GetAPIVersion() == "v1" {
//		//		table.Rows[0].Cells[0] = strings.ToLower(unstruct.GetKind()) + "/" + unstruct.GetName()
//		//	} else {
//		//		table.Rows[0].Cells[0] = strings.ToLower(unstruct.GetKind()) + "." + strings.Split(unstruct.GetAPIVersion(), "/")[0] + "/" + unstruct.GetName()
//		//	}
//		//} else {
//		//	table.Rows[0].Cells[0] = unstruct.GetName()
//		//}
//		//if unstruct.GetNamespace() != "" {
//		//	namespaceRaw := metav1.TableRow{}
//		//	namespaceRaw.Cells = append(namespaceRaw.Cells, unstruct.GetNamespace())
//		//	namespaceRaw.Cells = append(namespaceRaw.Cells, table.Rows[0].Cells...)
//		//	table.Rows[0] = namespaceRaw
//		//	columnWithNamespace := []metav1.TableColumnDefinition{}
//		//	columnWithNamespace = append(columnWithNamespace, metav1.TableColumnDefinition{Name: "NAMESPACE"})
//		//	columnWithNamespace = append(columnWithNamespace, table.ColumnDefinitions...)
//		//	table.ColumnDefinitions = columnWithNamespace
//		//}
//
//		return table
//
// }
func NewKoffCommand() *KoffCommand {
	koff := &KoffCommand{}
	koff.InitializeSchema()
	koff.InitializeTableGenerator()
	koff.Table = metav1.Table{}
	return koff
}

type KoffCommand struct {
	Kind           string
	NoHeaders      bool
	Namespace      string
	Wide           bool
	ShowLabels     bool
	SingleResource bool
	Items          []unstructured.UnstructuredList
	Output         bytes.Buffer
	CurrentKind    string
	LastKind       string
	Schema         *runtime.Scheme
	Printer        cliprint.ResourcePrinter
	Table          metav1.Table
	TableGenerator *printers.HumanReadableGenerator
	CRD            *apiextensionsv1.CustomResourceDefinition
	FromInput      bool
	ShowKind       bool
	ShowNamespace  bool
}

func (Koff *KoffCommand) InitializeTableGenerator() {
	Koff.TableGenerator = printers.NewTableGenerator()
	AddMissingHandlers(Koff.TableGenerator)
	printersinternal.AddHandlers(Koff.TableGenerator)
	buildprinters.AddBuildOpenShiftHandlers(Koff.TableGenerator)
	appsv1printer.AddAppsOpenShiftHandlers(Koff.TableGenerator)
	authorizationprinters.AddAuthorizationOpenShiftHandler(Koff.TableGenerator)
	imageprinters.AddImageOpenShiftHandlers(Koff.TableGenerator)
}

func (Koff *KoffCommand) InitializeSchema() {
	Koff.Schema = runtime.NewScheme()
	schemeBuilder := runtime.SchemeBuilder{
		metav1beta1.AddMetaToScheme,
		corev1.AddToScheme,
		apiregistration.AddToScheme,
	}
	_ = addAdmissionRegistrationTypes(Koff.Schema)
	_ = addApiServerInternalTypes(Koff.Schema)
	_ = addApiRegistrationTypes(Koff.Schema)
	_ = addAppsTypes(Koff.Schema)
	_ = addAppsV1Types(Koff.Schema)
	_ = addAuthorizationTypes(Koff.Schema)
	_ = addAutoscalingTypes(Koff.Schema)
	_ = addBatchTypes(Koff.Schema)
	_ = addBuildTypes(Koff.Schema)
	_ = addCertificatesTypes(Koff.Schema)
	_ = addCoordinationTypes(Koff.Schema)
	_ = addDiscoveryTypes(Koff.Schema)
	_ = addFlowControlTypes(Koff.Schema)
	_ = addFlowControlV1B2Types(Koff.Schema)
	_ = addImageTypes(Koff.Schema)
	_ = addNetworkingTypes(Koff.Schema)
	_ = addNodeTypes(Koff.Schema)
	_ = addPolicyV1Types(Koff.Schema)
	_ = addPolicyV1B1Types(Koff.Schema)
	_ = addResourceV1A2Types(Koff.Schema)
	_ = addRBACTypes(Koff.Schema)
	_ = addSchedulingTypes(Koff.Schema)
	_ = addStorageV1Types(Koff.Schema)
	_ = addStorageV1B1Types(Koff.Schema)
	utilruntime.Must(schemeBuilder.AddToScheme(Koff.Schema))
}
