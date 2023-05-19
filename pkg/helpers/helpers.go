package helpers

// specific labels https://github.com/seans3/kubernetes/blob/6108dac6708c026b172f3928e137c206437791da/pkg/printers/internalversion/printers_test.go#L1979
import (

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	//"k8s.io/kubernetes/pkg/apis/rbac"
	//rbac "k8s.io/api/rbac/v1"

	// "k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/apimachinery/pkg/api/meta"

	//runtime "k8s.io/apimachinery/pkg/runtime"
	//utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gmeghnag/koff/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	//core "k8s.io/kubernetes/pkg/apis/core"
	//ocpinternal "github.com/openshift/openshift-apiserver/pkg/apps/printers/internalversion"
	// cliprint "k8s.io/cli-runtime/pkg/printers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	//
)

func GetAge(resourcefilePath string, resourceCreationTimeStamp metav1.Time) string {
	ResourceFile, _ := os.Stat(resourcefilePath)
	t2 := ResourceFile.ModTime()
	diffTime := t2.Sub(resourceCreationTimeStamp.Time).String()
	d, _ := time.ParseDuration(diffTime)
	return FormatDiffTime(d)

}
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

func GetFromJsonPath(data interface{}, jsonPathTemplate string) string {
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
					Koff.GetArgs[normalizedResourceAlias] = make(map[string]struct{})
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
			resourceType = strings.SplitN(resourceType, ".", 1)[0]
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
			_crd := &apiextensionsv1.CustomResourceDefinition{Spec: crd.Spec}
			return strings.ToLower(_crd.Spec.Names.Kind), nil
		}
		resourceType, _, err := RetrieveKindGroupFromCRDS(koff, alias)
		if err == nil {
			return resourceType, nil
		}
	}
	return alias, fmt.Errorf("Alias \"%s\" not identified as any known resource or custom resource.", alias)
}

func RetrieveKindGroup(koff *types.KoffCommand, alias string) (string, string, error) {
	//if strings.Contains(alias, ".") {
	//	resourceKindAndGroup := strings.SplitN(alias, ".", 1)
	//	return resourceKindAndGroup[0], resourceKindAndGroup[1], nil
	//}

	value, ok := koff.KnownResources[alias]
	if ok {
		klog.V(3).Info("INFO ", fmt.Sprintf("found alias \"%s\" in known-resources.yaml", alias))
		resourceName := value["name"].(string)
		resourceGroup := value["group"].(string)
		return resourceName, resourceGroup, nil
	}
	klog.V(3).Info("INFO ", fmt.Sprintf("No internal resource found with name or alias \"%s\"", alias))
	return alias, "", fmt.Errorf("No internal resource found with name or alias \"%s\"", alias)
}
func RetrieveKindGroupFromCRDS(koff *types.KoffCommand, alias string) (string, string, error) {
	home, _ := os.UserHomeDir()
	crdsPath := home + "/.koff/customresourcedefinitions/"

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
			continue
		}
		koff.AliasToCrd[strings.ToLower(_crd.Spec.Names.Kind)] = apiextensionsv1.CustomResourceDefinition{Spec: _crd.Spec}
		if strings.ToLower(_crd.Spec.Names.Kind) == alias || strings.ToLower(_crd.Spec.Names.Plural) == alias || strings.ToLower(_crd.Spec.Names.Singular) == alias || StringInSlice(alias, _crd.Spec.Names.ShortNames) || _crd.Spec.Names.Singular+"."+_crd.Spec.Group == alias {
			koff.AliasToCrd[alias] = apiextensionsv1.CustomResourceDefinition{Spec: _crd.Spec}
			klog.V(4).Info("INFO ", fmt.Sprintf("Alias  \"%s\" found in path \"%s\".", alias, crdYamlPath))
			return strings.ToLower(_crd.Spec.Names.Kind), _crd.Spec.Group, nil
		}
		klog.V(5).Info("INFO ", fmt.Sprintf("Alias \"%s\" not found in path \"%s\".", alias, crdYamlPath))
	}
	klog.V(4).Info("INFO ", fmt.Sprintf("No customResource found with name or alias \"%s\" in path: \"%s\".", alias, crdsPath))
	return alias, "", fmt.Errorf("No customResource found with name or alias \"%s\"in path: \"%s\".", alias, crdsPath)
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Generate random alphanumeric string
// https://stackoverflow.com/a/31832326
var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "1234567890"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytes(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), err
}

// CONSTS
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// VARS
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// FUNCS
func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandString(length int) string {
	return StringWithCharset(length, charset)
}
