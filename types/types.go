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
	projectprinters "github.com/openshift/openshift-apiserver/pkg/project/printers/internalversion"
	quotaprinters "github.com/openshift/openshift-apiserver/pkg/quota/printers/internalversion"
	routeprinters "github.com/openshift/openshift-apiserver/pkg/route/printers/internalversion"
	securityprinters "github.com/openshift/openshift-apiserver/pkg/security/printers/internalversion"
	templateprinters "github.com/openshift/openshift-apiserver/pkg/template/printers/internalversion"

	//core "k8s.io/kubernetes/pkg/apis/core"
	//ocpinternal "github.com/openshift/openshift-apiserver/pkg/apps/printers/internalversion"

	"k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"

	// cliprint "k8s.io/cli-runtime/pkg/printers"

	runtime "k8s.io/apimachinery/pkg/runtime"
	cliprint "k8s.io/cli-runtime/pkg/printers"
)

func NewKoffCommand() *KoffCommand {
	koff := &KoffCommand{}
	koff.InitializeSchema()
	koff.UnstructuredList = UnstructuredList{Kind: "List", ApiVersion: "v1", Items: []unstructured.Unstructured{}}
	koff.InitializeTableGenerator()
	koff.Table = metav1.Table{}
	koff.GetArgs = make(map[string]map[string]struct{})
	koff.AliasToCrd = make(map[string]apiextensionsv1.CustomResourceDefinition)
	koff.ArgPresent = make(map[string]bool)
	return koff
}

type KoffCommand struct {
	NoHeaders         bool
	Namespace         string
	Wide              bool
	ShowLabels        bool
	SingleResource    bool
	UnstructuredList  UnstructuredList
	Output            bytes.Buffer
	CurrentKind       string
	LastKind          string
	Schema            *runtime.Scheme
	Printer           cliprint.ResourcePrinter
	Table             metav1.Table
	TableGenerator    *printers.HumanReadableGenerator
	CRD               *apiextensionsv1.CustomResourceDefinition
	FromInput         bool
	ShowKind          bool
	ShowNamespace     bool
	ShowManagedFields bool
	OutputFormat      string
	GetArgs           map[string]map[string]struct{}
	AliasToCrd        map[string]apiextensionsv1.CustomResourceDefinition
	ArgPresent        map[string]bool
}

type UnstructuredList struct {
	ApiVersion string                      `json:"apiVersion"`
	Kind       string                      `json:"kind"`
	Items      []unstructured.Unstructured `json:"items"`
}

func (Koff *KoffCommand) InitializeTableGenerator() {
	Koff.TableGenerator = printers.NewTableGenerator()
	AddMissingHandlers(Koff.TableGenerator)
	printersinternal.AddHandlers(Koff.TableGenerator)
	buildprinters.AddBuildOpenShiftHandlers(Koff.TableGenerator)
	appsv1printer.AddAppsOpenShiftHandlers(Koff.TableGenerator)
	authorizationprinters.AddAuthorizationOpenShiftHandler(Koff.TableGenerator)
	imageprinters.AddImageOpenShiftHandlers(Koff.TableGenerator)
	projectprinters.AddProjectOpenShiftHandlers(Koff.TableGenerator)
	quotaprinters.AddQuotaOpenShiftHandler(Koff.TableGenerator)
	securityprinters.AddSecurityOpenShiftHandler(Koff.TableGenerator)
	routeprinters.AddRouteOpenShiftHandlers(Koff.TableGenerator)
	templateprinters.AddTemplateOpenShiftHandlers(Koff.TableGenerator)
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
	_ = addProjectV1Types(Koff.Schema)
	_ = addQuotaV1Types(Koff.Schema)
	_ = addResourceV1A2Types(Koff.Schema)
	_ = addRouteV1Types(Koff.Schema)
	_ = addRBACTypes(Koff.Schema)
	_ = addSchedulingTypes(Koff.Schema)
	_ = addSecurityV1Types(Koff.Schema)
	_ = addStorageV1Types(Koff.Schema)
	_ = addStorageV1B1Types(Koff.Schema)
	_ = addTemplateV1Types(Koff.Schema)
	utilruntime.Must(schemeBuilder.AddToScheme(Koff.Schema))
}

type Context struct {
	Id        string `json:"id"`
	Path      string `json:"path"`
	InUse     string `json:"inUse"`
	Namespace string `json:"namespace"`
}

type Config struct {
	Id       string    `json:"id,omitempty"`
	Contexts []Context `json:"contexts,omitempty"`
	InUse    InUse     `json:"inUse,omitempty"`
}

type InUse struct {
	Path      string `json:"path"`
	IsBundle  bool   `json:"isBudle"`
	Namespace string `json:"namespace"`
}
