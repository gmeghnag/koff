package types

// specific labels https://github.com/seans3/kubernetes/blob/6108dac6708c026b172f3928e137c206437791da/pkg/printers/internalversion/printers_test.go#L1979
import (

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"
	"os"
	"strconv"
	"time"

	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	nodeapi "k8s.io/kubernetes/pkg/apis/node"

	appsv1 "github.com/openshift/openshift-apiserver/pkg/apps/apis/apps"
	imagev1 "github.com/openshift/openshift-apiserver/pkg/image/apis/image"
	corev1 "k8s.io/api/core/v1"

	authorizationv1 "github.com/openshift/api/authorization/v1"
	//"k8s.io/kubernetes/pkg/apis/rbac"
	//rbac "k8s.io/api/rbac/v1"

	"k8s.io/kubernetes/pkg/apis/coordination"
	"k8s.io/kubernetes/pkg/apis/networking"
	"k8s.io/kubernetes/pkg/apis/rbac"

	storage "k8s.io/kubernetes/pkg/apis/storage"

	// "k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/apimachinery/pkg/api/meta"

	//runtime "k8s.io/apimachinery/pkg/runtime"
	//utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"github.com/openshift/openshift-apiserver/pkg/build/apis/build"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	//core "k8s.io/kubernetes/pkg/apis/core"
	//ocpinternal "github.com/openshift/openshift-apiserver/pkg/apps/printers/internalversion"

	"k8s.io/kubernetes/pkg/apis/core"

	// cliprint "k8s.io/cli-runtime/pkg/printers"
	runtime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	//
	admissionregistration "k8s.io/kubernetes/pkg/apis/admissionregistration"
	apiserverinternal "k8s.io/kubernetes/pkg/apis/apiserverinternal"
	apps "k8s.io/kubernetes/pkg/apis/apps"
	autoscaling "k8s.io/kubernetes/pkg/apis/autoscaling"
	batch "k8s.io/kubernetes/pkg/apis/batch"
	certificates "k8s.io/kubernetes/pkg/apis/certificates"
	discovery "k8s.io/kubernetes/pkg/apis/discovery"
	flowcontrol "k8s.io/kubernetes/pkg/apis/flowcontrol"
	policy "k8s.io/kubernetes/pkg/apis/policy"
	resource "k8s.io/kubernetes/pkg/apis/resource"
	scheduling "k8s.io/kubernetes/pkg/apis/scheduling"
)

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

func NewKoffCommand() *KoffCommand {
	koff := &KoffCommand{}
	koff.InitializeSchema()
	koff.InitializeTableGenerator()
	koff.Table = metav1.Table{}
	return koff
}

func rawObjectToRuntimeObject(rawObject []byte, schema *runtime.Scheme) runtime.Object {
	codec := serializer.NewCodecFactory(schema)
	decode := codec.UniversalDeserializer()
	obj, _, err := decode.Decode([]byte(rawObject), nil, nil)
	if err != nil {
		//fmt.Println("loglevel2", err)
	}
	switch obj.(type) {
	case *admissionregistration.MutatingWebhookConfiguration:
		return &admissionregistration.MutatingWebhookConfiguration{}
	case *admissionregistration.ValidatingAdmissionPolicyBinding:
		return &admissionregistration.ValidatingAdmissionPolicyBinding{}
	case *admissionregistration.ValidatingWebhookConfiguration:
		return &admissionregistration.ValidatingWebhookConfiguration{}
	case *admissionregistration.ValidatingAdmissionPolicy:
		return &admissionregistration.ValidatingAdmissionPolicy{}
	case *apiregistration.APIService:
		return &apiregistration.APIService{}
	case *apiserverinternal.StorageVersion:
		return &apiserverinternal.StorageVersion{}
	case *apps.StatefulSet:
		return &apps.StatefulSet{}
	case *apps.ReplicaSet:
		return &apps.ReplicaSet{}
	case *apps.Deployment:
		return &apps.Deployment{}
	case *apps.DaemonSet:
		return &apps.DaemonSet{}
	case *apps.ControllerRevision:
		return &apps.ControllerRevision{}
	case *appsv1.DeploymentConfig:
		return &appsv1.DeploymentConfig{}
	case *authorizationv1.ClusterRole:
		return &rbac.ClusterRole{}
	case *authorizationv1.ClusterRoleBinding:
		return &rbac.ClusterRoleBinding{}
	case *authorizationv1.RoleBindingRestriction:
		return &authorizationv1.RoleBindingRestriction{}
	case *authorizationv1.SubjectRulesReview:
		return &authorizationv1.SubjectRulesReview{}
	case *autoscaling.HorizontalPodAutoscaler:
		return &autoscaling.HorizontalPodAutoscaler{}
	case *autoscaling.Scale:
		return &autoscaling.Scale{}
	case *batch.CronJob:
		return &batch.CronJob{}
	case *batch.Job:
		return &batch.Job{}
	case *build.Build:
		return &build.Build{}
	case *build.BuildConfig:
		return &build.BuildConfig{}
	case *certificates.CertificateSigningRequest:
		return &certificates.CertificateSigningRequest{}
	case *certificates.ClusterTrustBundle:
		return &certificates.ClusterTrustBundle{}
	case *coordination.Lease:
		return &coordination.Lease{}
	case *corev1.Pod:
		return &core.Pod{}
	case *corev1.PodTemplate:
		return &core.PodTemplate{}
	case *corev1.ReplicationController:
		return &core.ReplicationController{}
	case *corev1.Service:
		return &core.Service{}
	case *corev1.Endpoints:
		return &core.Endpoints{}
	case *corev1.Namespace:
		return &core.Namespace{}
	case *corev1.Secret:
		return &core.Secret{}
	case *corev1.ServiceAccount:
		return &core.ServiceAccount{}
	case *corev1.Node:
		return &core.Node{}
	case *corev1.PersistentVolume:
		return &core.PersistentVolume{}
	case *corev1.PersistentVolumeClaim:
		return &core.PersistentVolumeClaim{}
	case *corev1.Event:
		return &core.Event{}
	case *corev1.ComponentStatus:
		return &core.ComponentStatus{}
	case *corev1.ConfigMap:
		return &core.ConfigMap{}
	case *corev1.ResourceQuota:
		return &core.ResourceQuota{}
	case *discovery.EndpointSlice:
		return &discovery.EndpointSlice{}
	case *flowcontrol.FlowSchema:
		return &flowcontrol.FlowSchema{}
	case *flowcontrol.PriorityLevelConfiguration:
		return &flowcontrol.PriorityLevelConfiguration{}
	case *imagev1.ImageStream:
		return &imagev1.ImageStream{}
	case *imagev1.ImageStreamTag:
		return &imagev1.ImageStreamTag{}
	case *networking.ClusterCIDR:
		return &networking.ClusterCIDR{}
	case *networking.IPAddress:
		return &networking.IPAddress{}
	case *networking.IngressClass:
		return &networking.IngressClass{}
	case *networking.Ingress:
		return &networking.Ingress{}
	case *networking.NetworkPolicy:
		return &networking.NetworkPolicy{}
	case *nodeapi.RuntimeClass:
		return &nodeapi.RuntimeClass{}
	case *policy.PodDisruptionBudget:
		return &policy.PodDisruptionBudget{}
	case *policy.PodSecurityPolicy:
		return &policy.PodSecurityPolicy{}
	case *resource.ResourceClass:
		return &resource.ResourceClass{}
	case *resource.ResourceClaim:
		return &resource.ResourceClaim{}
	case *resource.ResourceClaimTemplate:
		return &resource.ResourceClaimTemplate{}
	case *resource.PodSchedulingContext:
		return &resource.PodSchedulingContext{}
	case *rbac.ClusterRole:
		return &rbac.ClusterRole{}
	case *rbac.ClusterRoleBinding:
		return &rbac.ClusterRoleBinding{}
	case *rbac.Role:
		return &rbac.Role{}
	case *rbac.RoleBinding:
		return &rbac.RoleBinding{}
	case *scheduling.PriorityClass:
		return &scheduling.PriorityClass{}
	case *storage.CSIStorageCapacity:
		return &storage.CSIStorageCapacity{}
	case *storage.StorageClass:
		return &storage.StorageClass{}
	case *storage.CSINode:
		return &storage.CSINode{}
	case *storage.CSIDriver:
		return &storage.CSIDriver{}
	case *storage.VolumeAttachment:
		return &storage.VolumeAttachment{}
	}
	//fmt.Println("RUNTIME UNKNOW")
	return &runtime.Unknown{}
}

func GetAge(resourcefilePath string, resourceCreationTimeStamp metav1.Time) string {
	ResourceFile, _ := os.Stat(resourcefilePath)
	t2 := ResourceFile.ModTime()
	diffTime := t2.Sub(resourceCreationTimeStamp.Time).String()
	d, _ := time.ParseDuration(diffTime)
	return FormatDiffTime(d)

}
func translateTimestamp(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return ShortHumanDuration(time.Now().Sub(timestamp.Time))
}
func ShortHumanDuration(d time.Duration) string {
	// Allow deviation no more than 2 seconds(excluded) to tolerate machine time
	// inconsistence, it can be considered as almost now.
	if seconds := int(d.Seconds()); seconds < -1 {
		return fmt.Sprintf("<invalid>")
	} else if seconds < 0 {
		return fmt.Sprintf("0s")
	} else if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if minutes := int(d.Minutes()); minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	} else if hours := int(d.Hours()); hours < 24 {
		return fmt.Sprintf("%dh", hours)
	} else if hours < 24*365 {
		return fmt.Sprintf("%dd", hours/24)
	}
	return fmt.Sprintf("%dy", int(d.Hours()/24/365))
}

func FormatDiffTime(diff time.Duration) string {
	if diff.Hours() > 48 {
		if diff.Hours() > 200000 {
			return "Unknown"
		}
		return strconv.Itoa(int(diff.Hours()/24)) + "d"
	}
	if diff.Hours() < 48 && diff.Hours() > 10 {
		var h float64
		h = diff.Minutes() / 60
		return strconv.Itoa(int(h)) + "h"
	}
	if diff.Minutes() > 60 {
		var hours float64
		hours = diff.Minutes() / 60
		remainMinutes := int(diff.Minutes()) % 60
		if remainMinutes > 0 {
			return strconv.Itoa(int(hours)) + "h" + strconv.Itoa(remainMinutes) + "m"
		}
		return strconv.Itoa(int(hours)) + "h"

	}
	if diff.Seconds() > 60 {
		var minutes float64
		minutes = diff.Seconds() / 60
		remainSeconds := int(diff.Seconds()) % 60
		if remainSeconds > 0 && diff.Minutes() < 4 {
			return strconv.Itoa(int(minutes)) + "m" + strconv.Itoa(remainSeconds) + "s"
		}
		return strconv.Itoa(int(minutes)) + "m"

	}
	return strconv.Itoa(int(diff.Seconds())) + "s"
}
