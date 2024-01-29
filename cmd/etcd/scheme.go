/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"

	authenticationv1 "k8s.io/api/authentication/v1"
	authenticationv1beta1 "k8s.io/api/authentication/v1beta1"

	authorizationv1 "k8s.io/api/authorization/v1"
	authorizationv1beta1 "k8s.io/api/authorization/v1beta1"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"

	certificatesv1 "k8s.io/api/certificates/v1"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"

	discoveryv1 "k8s.io/api/discovery/v1"
	flowcontrolv1beta1 "k8s.io/api/flowcontrol/v1beta1"

	corev1 "k8s.io/api/core/v1"

	eventsv1beta1 "k8s.io/api/events/v1beta1"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"

	imagepolicyv1alpha1 "k8s.io/api/imagepolicy/v1alpha1"

	coordinationv1 "k8s.io/api/coordination/v1"
	networkingv1 "k8s.io/api/networking/v1"

	quotav1 "github.com/openshift/api/quota/v1"
	securityv1 "github.com/openshift/api/security/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"

	schedulingv1alpha1 "k8s.io/api/scheduling/v1alpha1"

	storagev1 "k8s.io/api/storage/v1"
	storagev1alpha1 "k8s.io/api/storage/v1alpha1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"

	//OCP API
	ocpappsv1 "github.com/openshift/api/apps/v1"
	ocpauthorizationv1 "github.com/openshift/api/authorization/v1"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	projectv1 "github.com/openshift/api/project/v1"
	routev1 "github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
)

var Scheme = runtime.NewScheme()
var Codecs = serializer.NewCodecFactory(Scheme)
var ParameterCodec = runtime.NewParameterCodec(Scheme)

func init() {
	v1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	AddToScheme(Scheme)
}

func AddToScheme(scheme *runtime.Scheme) {
	templatev1.AddToScheme(scheme)
	imagev1.AddToScheme(scheme)
	buildv1.AddToScheme(scheme)
	ocpappsv1.AddToScheme(scheme)
	projectv1.AddToScheme(scheme)
	routev1.AddToScheme(scheme)
	quotav1.AddToScheme(scheme)
	securityv1.AddToScheme(scheme)
	ocpauthorizationv1.AddToScheme(scheme)
	admissionv1beta1.AddToScheme(scheme)
	apiregistrationv1.AddToScheme(scheme)
	apiextensionsv1.AddToScheme(scheme)
	apiextensionsv1beta1.AddToScheme(scheme)
	admissionregistrationv1alpha1.AddToScheme(scheme)
	admissionregistrationv1beta1.AddToScheme(scheme)

	appsv1.AddToScheme(scheme)
	appsv1beta1.AddToScheme(scheme)
	appsv1beta2.AddToScheme(scheme)

	authenticationv1.AddToScheme(scheme)
	authenticationv1beta1.AddToScheme(scheme)

	authorizationv1.AddToScheme(scheme)
	authorizationv1beta1.AddToScheme(scheme)

	autoscalingv1.AddToScheme(scheme)
	autoscalingv2beta1.AddToScheme(scheme)

	batchv1.AddToScheme(scheme)
	batchv1beta1.AddToScheme(scheme)

	certificatesv1.AddToScheme(scheme)
	certificatesv1beta1.AddToScheme(scheme)

	corev1.AddToScheme(scheme)
	coordinationv1.AddToScheme(scheme)
	discoveryv1.AddToScheme(scheme)
	eventsv1beta1.AddToScheme(scheme)

	extensionsv1beta1.AddToScheme(scheme)
	flowcontrolv1beta1.AddToScheme(scheme)

	imagepolicyv1alpha1.AddToScheme(scheme)

	networkingv1.AddToScheme(scheme)

	policyv1.AddToScheme(scheme)
	policyv1beta1.AddToScheme(scheme)

	rbacv1.AddToScheme(scheme)
	rbacv1alpha1.AddToScheme(scheme)
	rbacv1beta1.AddToScheme(scheme)

	schedulingv1alpha1.AddToScheme(scheme)

	storagev1.AddToScheme(scheme)
	storagev1alpha1.AddToScheme(scheme)
	storagev1beta1.AddToScheme(scheme)
}
