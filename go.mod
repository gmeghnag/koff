module github.com/gmeghnag/koff

go 1.23

toolchain go1.23.0

require (
	github.com/coreos/go-semver v0.3.1
	github.com/openshift/api v0.0.0-20241126051952-8f4a33eac08f
	github.com/openshift/openshift-apiserver v0.0.0-alpha.0.0.20230503155233-aac3dd5bf054
	github.com/schollz/progressbar/v3 v3.17.1
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	go.etcd.io/bbolt v1.3.11
	go.etcd.io/etcd/api/v3 v3.5.17
	go.etcd.io/etcd/etcdutl/v3 v3.5.17
	golang.org/x/crypto v0.29.0
	golang.org/x/term v0.26.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.31.3
	k8s.io/apiextensions-apiserver v0.31.3
	k8s.io/apimachinery v0.31.3
	k8s.io/cli-runtime v0.31.3
	k8s.io/client-go v0.31.3
	k8s.io/klog/v2 v2.130.1
	k8s.io/kube-aggregator v0.31.3
	k8s.io/kubernetes v1.31.3
	k8s.io/kubernetes/v1_28 v0.0.0-00010101000000-000000000000
	k8s.io/kubernetes/v1_30 v0.0.0-00010101000000-000000000000
	sigs.k8s.io/yaml v1.4.0
)

replace (
	// - networking.ClusterCIDR is deprecated as of v1.29.0:
	//   https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.29.md#deprecation
	// - policy.PodSecurityPolicy also removed as of v1.29.0 sources:
	//   https://kubernetes.io/blog/2021/04/06/podsecuritypolicy-deprecation-past-present-and-future/
	// so replace it under a versioned dir to keep pre-v1.29.0 resources available:
	k8s.io/kubernetes/v1_28 => k8s.io/kubernetes v1.28.15
	// - also resource.ResourceClass no longer available in 1.31:
	// so also replace it under a versioned dir to keep pre-v1.31.0 resources available:
	k8s.io/kubernetes/v1_30 => k8s.io/kubernetes v1.30.7
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/gnostic v0.7.0 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/openshift/library-go v0.0.0-20241125211829-543f84d25447 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.60.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xiang90/probing v0.0.0-20221125231312-a49e3df8f510 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.17 // indirect
	go.etcd.io/etcd/client/v2 v2.305.17 // indirect
	go.etcd.io/etcd/client/v3 v3.5.17 // indirect
	go.etcd.io/etcd/pkg/v3 v3.5.17 // indirect
	go.etcd.io/etcd/raft/v3 v3.5.17 // indirect
	go.etcd.io/etcd/server/v3 v3.5.17 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.57.0 // indirect
	go.opentelemetry.io/otel v1.32.0 // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/grpc v1.68.0 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.31.3 // indirect
	k8s.io/component-base v0.31.3 // indirect
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f // indirect
	k8s.io/utils v0.0.0-20241104163129-6fe5fd82f078 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.3 // indirect
)
