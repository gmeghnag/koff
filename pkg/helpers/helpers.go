package helpers

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gmeghnag/koff/types"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

func TranslateTimestamp(timestamp metav1.Time) string {
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

func GetFromJsonPath(data any, jsonPathTemplate string) string {
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

func ParseGetArgs(Koff *types.KoffCommand, args []string) error {
	var _args []string
	for _, arg := range args {
		_args = append(_args, strings.ToLower(arg))
	}
	args = _args
	if len(args) == 1 && !strings.Contains(args[0], "/") {
		if strings.Contains(args[0], ",") {
			resourcesTypes := strings.Split(strings.TrimPrefix(strings.TrimSuffix(args[0], ","), ","), ",")
			for _, resourceType := range resourcesTypes {
				if strings.Contains(resourceType, ".") {
					resourceType = strings.SplitN(resourceType, ".", 2)[0]
				}
				normalizedResourceAlias, err := normalizeResourceAlias(Koff, resourceType)
				if err == nil {
					Koff.GetArgs[normalizedResourceAlias] = make(map[string]struct{})
				} else {
					Koff.ArgPresent[resourceType] = false
					Koff.GetArgs[resourceType] = make(map[string]struct{})
				}

			}
		} else {
			resourceType := args[0]
			if strings.Contains(args[0], ".") {
				resourceType = strings.SplitN(args[0], ".", 2)[0]
			}
			normalizedResourceAlias, err := normalizeResourceAlias(Koff, resourceType)
			if err == nil {
				Koff.GetArgs[normalizedResourceAlias] = make(map[string]struct{})
			} else {
				Koff.ArgPresent[resourceType] = false
				Koff.GetArgs[resourceType] = make(map[string]struct{})
			}
		}
	} else if len(args) > 0 && strings.Contains(args[0], "/") {
		if len(args) == 1 {
			Koff.SingleResource = true
		}
		for _, arg := range args {
			if strings.Contains(arg, "/") {
				resource := strings.Split(arg, "/")
				resourceType, resourceName := resource[0], resource[1]
				if strings.Contains(resourceType, ".") {
					resourceType = strings.SplitN(resourceType, ".", 2)[0]
				}
				normalizedResourceAlias, err := normalizeResourceAlias(Koff, resourceType)
				if err == nil {
					_, ok := Koff.GetArgs[normalizedResourceAlias]
					if !ok {
						Koff.GetArgs[normalizedResourceAlias] = make(map[string]struct{})
					}
					Koff.GetArgs[normalizedResourceAlias][resourceName] = struct{}{}
				} else {
					Koff.ArgPresent[resourceType] = false
					Koff.GetArgs[resourceType] = make(map[string]struct{})
					Koff.GetArgs[resourceType][resourceName] = struct{}{}
				}
			} else {
				return fmt.Errorf("there is no need to specify a resource type as a separate argument when passing arguments in resource/name form (e.g. 'oc get resource/<resource_name>' instead of 'oc get resource resource/<resource_name>'")
			}
		}
	} else if len(args) > 1 && !strings.Contains(args[0], "/") {
		resourceType := args[0]
		if strings.Contains(resourceType, ".") {
			resourceType = strings.SplitN(resourceType, ".", 2)[0]
		}
		normalizedResourceAlias, err := normalizeResourceAlias(Koff, resourceType)
		if err == nil {
			Koff.GetArgs[normalizedResourceAlias] = make(map[string]struct{})
		} else {
			Koff.ArgPresent[resourceType] = false
			Koff.GetArgs[resourceType] = make(map[string]struct{})
		}
		if len(args[0:]) == 2 {
			Koff.SingleResource = true
		}
		for _, resourceName := range args[1:] {
			if strings.Contains(resourceName, "/") {
				return fmt.Errorf("there is no need to specify a resource type as a separate argument when passing arguments in resource/name form (e.g. 'oc get resource/<resource_name>' instead of 'oc get resource resource/<resource_name>'")
			}
			Koff.GetArgs[normalizedResourceAlias][resourceName] = struct{}{}
		}
	}
	return nil
}

func normalizeResourceAlias(koff *types.KoffCommand, alias string) (string, error) {
	value, ok := koff.KnownResources[alias]
	if ok {
		klog.V(3).Info("INFO ", fmt.Sprintf("Alias \"%s\" is a known resource.", alias))
		resourceType := value["name"].(string)
		return resourceType, nil
	} else {
		klog.V(3).Info("INFO ", fmt.Sprintf("Alias \"%s\" resource not known.", alias))
		crd, ok := koff.AliasToCrd[alias]
		if ok {
			_crd := &apiextensionsv1beta1.CustomResourceDefinition{Spec: crd.Spec}
			return strings.ToLower(_crd.Spec.Names.Kind), nil
		}
		resourceType, _, err := RetrieveKindGroupFromCRDS(koff, alias)
		if err == nil {
			return resourceType, nil
		}
	}
	return alias, fmt.Errorf("alias \"%s\" not identified as any known resource or custom resource", alias)
}

func RetrieveKindGroupFromCRDS(koff *types.KoffCommand, alias string) (string, string, error) {
	if koff.IsEtcdDb {
		aliasFields, ok := koff.EtcdAliasToCrdKubeKey[alias]
		if ok {
			return strings.ToLower(aliasFields.Kind), aliasFields.Group, nil
		}
		return alias, "", fmt.Errorf("no customResource found with name or alias \"%s\"in etcd db: ", alias)
	}
	loadCRDsFromDisk(koff)
	if crd, ok := koff.AliasToCrd[alias]; ok {
		return strings.ToLower(crd.Spec.Names.Kind), crd.Spec.Group, nil
	}
	return alias, "", fmt.Errorf("no customResource found with name or alias \"%s\"", alias)
}

// loadCRDsFromDisk reads every CRD under ~/.koff/customresourcedefinitions once
// and registers all of its aliases (kind, plural, singular, short names and
// singular.group) into koff.AliasToCrd. Subsequent calls are no-ops.
func loadCRDsFromDisk(koff *types.KoffCommand) {
	if koff.CRDsLoaded {
		return
	}
	koff.CRDsLoaded = true

	home, err := os.UserHomeDir()
	if err != nil {
		klog.V(1).ErrorS(err, "ERROR resolving home directory")
		return
	}
	crdsPath := filepath.Join(home, ".koff", "customresourcedefinitions")
	crds, err := os.ReadDir(crdsPath)
	if err != nil {
		klog.V(4).Info("INFO ", fmt.Sprintf("could not read CRD directory %q: %v", crdsPath, err))
		return
	}
	for _, f := range crds {
		crdByte, err := os.ReadFile(filepath.Join(crdsPath, f.Name()))
		if err != nil {
			continue
		}
		crd := &apiextensionsv1beta1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(crdByte, crd); err != nil {
			continue
		}
		entry := apiextensionsv1beta1.CustomResourceDefinition{Spec: crd.Spec}
		aliases := []string{
			strings.ToLower(crd.Spec.Names.Kind),
			strings.ToLower(crd.Spec.Names.Plural),
			strings.ToLower(crd.Spec.Names.Singular),
			crd.Spec.Names.Singular + "." + crd.Spec.Group,
		}
		aliases = append(aliases, crd.Spec.Names.ShortNames...)
		for _, a := range aliases {
			if a == "" || a == "." {
				continue
			}
			koff.AliasToCrd[strings.ToLower(a)] = entry
		}
	}
}

// etcdCoreStoragePath maps a core-group resource's plural name to the path
// segment actually used under /kubernetes.io/ in etcd, for the historical cases
// where the storage name differs from the plural:
//   - nodes are stored under "minions" (legacy name)
//   - services / endpoints live under "services/specs" and "services/endpoints"
var etcdCoreStoragePath = map[string]string{
	"nodes":     "minions",
	"services":  "services/specs",
	"endpoints": "services/endpoints",
}

func EtcdPrefixFromAlias(koff *types.KoffCommand, alias string, resourceName string) (string, error) {
	var etcdPrefixResource string
	value, ok := koff.KnownResources[alias]
	if ok {
		klog.V(3).Info("INFO ", fmt.Sprintf("Alias \"%s\" is a known resource.", alias))
		resourceNamePlural := value["plural"].(string)
		resourceGroup := value["group"].(string)
		resourceNamespaced := value["namespaced"].(bool)
		if strings.HasSuffix(resourceGroup, "openshift.io") {
			etcdPrefixResource = "/openshift.io/" + resourceGroup + "/" + resourceNamePlural
		} else {
			if resourceGroup == "core" || resourceGroup == "events.k8s.io" || resourceGroup == "apps" {
				segment := resourceNamePlural
				if override, ok := etcdCoreStoragePath[resourceNamePlural]; ok {
					segment = override
				}
				etcdPrefixResource = "/kubernetes.io/" + segment
			} else {
				etcdPrefixResource = "/kubernetes.io/" + resourceGroup + "/" + resourceNamePlural
			}
		}
		if !koff.AllNamespaces {
			if resourceNamespaced {
				etcdPrefixResource += "/" + koff.Namespace
			}
			if resourceName != "" {
				etcdPrefixResource += "/" + resourceName
			}
		}
		return etcdPrefixResource, nil
	} else {
		aliasFields, ok := koff.EtcdAliasToCrdKubeKey[alias]
		if ok {
			if koff.AllNamespaces {
				etcdPrefixResource = "/kubernetes.io/" + aliasFields.Group + "/" + aliasFields.Plural
			} else {
				if aliasFields.Namespaced {
					etcdPrefixResource = "/kubernetes.io/" + aliasFields.Group + "/" + aliasFields.Plural + "/" + koff.Namespace
				} else {
					etcdPrefixResource = "/kubernetes.io/" + aliasFields.Group + "/" + aliasFields.Plural
				}
				if resourceName != "" {
					etcdPrefixResource += "/" + resourceName
				}
			}
			return etcdPrefixResource, nil
		}
	}
	return "", fmt.Errorf("alias \"%s\" not identified as any known resource or custom resource", alias)
}
