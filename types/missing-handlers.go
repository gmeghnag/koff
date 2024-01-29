package types

import (
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/kubernetes/pkg/printers"
)

func AddMissingHandlers(h printers.PrintHandler) {
	apiServiceColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name"},
		{Name: "Service", Type: "string"},
		{Name: "Available", Type: "string"},
		{Name: "Age", Type: "string"},
	}

	customResourceDefinitionColumnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name"},
		{Name: "Ceated At", Type: "string"},
	}

	_ = h.TableHandler(apiServiceColumnDefinitions, printAPIService)
	_ = h.TableHandler(customResourceDefinitionColumnDefinitions, printCustomResourceDefinitionv1)
	_ = h.TableHandler(customResourceDefinitionColumnDefinitions, printCustomResourceDefinitionv1beta1)
}

func printAPIService(obj *apiregistrationv1.APIService, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	service := "Local"
	if obj.Spec.Service != nil {
		service = obj.Spec.Service.Namespace + "/" + obj.Spec.Service.Name
	}
	available := "Unknown"
	for _, condition := range obj.Status.Conditions {
		if condition.Type == "Available" {
			available = string(condition.Status)
			if available != "True" {
				available = string(condition.Status) + " (" + condition.Reason + ")"
			}
			break
		}
	}
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, obj.Name, service, available, "")
	return []metav1.TableRow{row}, nil
}

func printCustomResourceDefinitionv1(obj *apiextensionsv1.CustomResourceDefinition, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	creationTimestamp := obj.GetCreationTimestamp()
	createdAt := creationTimestamp.UTC().Format(time.RFC3339Nano)
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, obj.Name, createdAt)
	return []metav1.TableRow{row}, nil
}

func printCustomResourceDefinitionv1beta1(obj *apiextensionsv1beta1.CustomResourceDefinition, options printers.GenerateOptions) ([]metav1.TableRow, error) {
	creationTimestamp := obj.GetCreationTimestamp()
	createdAt := creationTimestamp.UTC().Format(time.RFC3339Nano)
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: obj},
	}
	row.Cells = append(row.Cells, obj.Name, createdAt)
	return []metav1.TableRow{row}, nil
}
