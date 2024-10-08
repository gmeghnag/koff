package get

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gmeghnag/koff/cmd/etcd"
	"github.com/gmeghnag/koff/types"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	//
)

const (
	StorageBinaryMediaType = "application/vnd.kubernetes.storagebinary"
	ProtobufMediaType      = "application/vnd.kubernetes.protobuf"
	YamlMediaType          = "application/yaml"
	JsonMediaType          = "application/json"
)

var formatOutput = "yaml"
var output bytes.Buffer
var stringToKey = make(map[string][]byte)

func inspectResources(koff *types.KoffCommand, dbPath string, keyName string) error {

	//h := crc32.New(crc32.MakeTable(crc32.Castagnoli))

	if err := koff.EtcdDb.View(func(tx *bolt.Tx) error {
		// check snapshot file integrity first
		var dbErrStrings []string
		for dbErr := range tx.Check() {
			dbErrStrings = append(dbErrStrings, dbErr.Error())
		}
		if len(dbErrStrings) > 0 {
			return fmt.Errorf("snapshot file integrity check failed. %d errors found.\n"+strings.Join(dbErrStrings, "\n"), len(dbErrStrings))
		}
		//c := tx.Cursor()
		//var out string
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("cannot get hash of bucket %s", string("key"))
		}
		var keys [][]byte
		b.ForEach(func(k, value []byte) error {
			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(value); err != nil {
				panic(err)
			}
			keys = append(keys, kv.Key)
			koff.KubeKeysToEtcdKeys[string(kv.Key)] = k
			if strings.HasPrefix(string(kv.Key), "/kubernetes.io/apiextensions.k8s.io/customresourcedefinitions/") {

			}

			// Sort the keys in lexicographical order

			if keyName != "" {
				if string(kv.Key) == keyName {
					unstruct := &unstructured.Unstructured{}
					err := unstruct.UnmarshalJSON(kv.Value)
					if err == nil {
						if formatOutput == "json" {
							data, _ := json.MarshalIndent(unstruct, "", "  ")
							data = append(data, '\n')
							fmt.Printf("%s", data)
							return nil
						} else if formatOutput == "yaml" {
							data, _ := yaml.Marshal(unstruct)
							fmt.Printf("%s", data)
						}

					}
					if formatOutput == "json" {
						_, err = etcd.DetectAndConvert(JsonMediaType, kv.Value, &output)
						if err != nil {
							panic(err)
						}
						err := unstruct.UnmarshalJSON(output.Bytes())
						if err == nil {
							data, _ := json.MarshalIndent(unstruct, "", "  ")
							data = append(data, '\n')
							fmt.Printf("%s", data)
							return nil
						}
					} else if formatOutput == "yaml" {
						_, err = etcd.DetectAndConvert(YamlMediaType, kv.Value, &output)
						if err != nil {
							panic(err)
						}
						fmt.Printf("%s", &output)
					}
					os.Exit(0)
				}
			} else {
				////fmt.Println(string(kv.Key))
			}
			return nil
		})
		sort.Slice(keys, func(i, j int) bool {
			return string(keys[i]) < string(keys[j])
		})
		//for _, key := range keys {
		//	fmt.Printf("%s\n", key)
		//}
		etcdKey, exists := koff.KubeKeysToEtcdKeys["/kubernetes.io/configmaps/openshift-config-managed/admin-gates"]
		if exists {
			fmt.Printf("Value for key '%s': %s\n", "/kubernetes.io/configmaps/openshift-config-managed/admin-gates", etcdKey)
			etcdvalue := b.Get(etcdKey)
			var kv_ mvccpb.KeyValue
			if err := kv_.Unmarshal(etcdvalue); err != nil {
				panic(err)
			}
			_, err := etcd.DetectAndConvert(JsonMediaType, kv_.Value, &output)
			unstruct := &unstructured.Unstructured{}
			err = unstruct.UnmarshalJSON(output.Bytes())
			if err == nil {
				data, _ := json.MarshalIndent(unstruct, "", "  ")
				data = append(data, '\n')
				fmt.Printf("%s", data)
				return nil
			}
		} else {
			fmt.Printf("Key '%s' not found\n", "/kubernetes.io/configmaps/openshift-config-managed/admin-gates")
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func populateCRDsFromEtcd(koff *types.KoffCommand, dbPath string, keyName string) error {
	if err := koff.EtcdDb.View(func(tx *bolt.Tx) error {
		// check snapshot file integrity first
		var dbErrStrings []string
		for dbErr := range tx.Check() {
			dbErrStrings = append(dbErrStrings, dbErr.Error())
		}
		if len(dbErrStrings) > 0 {
			return fmt.Errorf("snapshot file integrity check failed. %d errors found.\n"+strings.Join(dbErrStrings, "\n"), len(dbErrStrings))
		}
		//c := tx.Cursor()
		//var out string
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("cannot get hash of bucket %s", string("key"))
		}
		var keys [][]byte
		b.ForEach(func(k, value []byte) error {
			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(value); err != nil {
				panic(err)
			}
			keys = append(keys, kv.Key)
			koff.KubeKeysToEtcdKeys[string(kv.Key)] = k
			if strings.HasPrefix(string(kv.Key), "/kubernetes.io/apiextensions.k8s.io/customresourcedefinitions/") {
				_crd := &apiextensionsv1.CustomResourceDefinition{}
				if err := yaml.Unmarshal([]byte(kv.Value), &_crd); err != nil {
				}
				namespaced := func(scope string) bool {
					return scope == "Namespaced"
				}(string(_crd.Spec.Scope))
				koff.EtcdAliasToCrdKubeKey[strings.ToLower(_crd.Spec.Names.Kind)] = types.AliasSubField{Kind: strings.ToLower(_crd.Spec.Names.Kind),
					Plural:     strings.ToLower(_crd.Spec.Names.Plural),
					KubeKey:    string(kv.Key),
					Group:      strings.ToLower(_crd.Spec.Group),
					Namespaced: namespaced}
				koff.EtcdAliasToCrdKubeKey[strings.ToLower(_crd.Spec.Names.Plural)] = types.AliasSubField{Kind: strings.ToLower(_crd.Spec.Names.Kind),
					Plural:     strings.ToLower(_crd.Spec.Names.Plural),
					KubeKey:    string(kv.Key),
					Group:      strings.ToLower(_crd.Spec.Group),
					Namespaced: namespaced}
				koff.EtcdAliasToCrdKubeKey[strings.ToLower(_crd.Spec.Names.Singular)] = types.AliasSubField{Kind: strings.ToLower(_crd.Spec.Names.Kind),
					Plural:     strings.ToLower(_crd.Spec.Names.Plural),
					KubeKey:    string(kv.Key),
					Group:      strings.ToLower(_crd.Spec.Group),
					Namespaced: namespaced}
				for _, shortName := range _crd.Spec.Names.ShortNames {
					koff.EtcdAliasToCrdKubeKey[strings.ToLower(shortName)] = types.AliasSubField{Kind: strings.ToLower(_crd.Spec.Names.Kind),
						Plural:     strings.ToLower(_crd.Spec.Names.Plural),
						KubeKey:    string(kv.Key),
						Group:      strings.ToLower(_crd.Spec.Group),
						Namespaced: namespaced}
				}
				koff.EtcdAliasToCrdKubeKey[_crd.Spec.Names.Singular+"."+_crd.Spec.Group] = types.AliasSubField{Kind: strings.ToLower(_crd.Spec.Names.Kind),
					Plural:     strings.ToLower(_crd.Spec.Names.Plural),
					KubeKey:    string(kv.Key),
					Group:      strings.ToLower(_crd.Spec.Group),
					Namespaced: namespaced}
			}
			return nil
		})
		sort.Slice(keys, func(i, j int) bool {
			return string(keys[i]) < string(keys[j])
		})
		// koff.KubeKeysToEtcdKeys non sortato valuta se creare un
		// array field per koff dove salvare keys -> [][]bytes
		//for _, key := range keys {
		//	koff.KubeKeysToEtcdKeys[string(key)] = stringToKey[string(key)]
		//}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func GetResourcesFromEtcd(koff *types.KoffCommand, etcdKeyPrefixesToCheck map[string]bool) error {
	if err := koff.EtcdDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("cannot get hash of bucket %s", string("key"))
		}
		b.ForEach(func(k, value []byte) error {
			var resourceToHandle bool
			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(value); err != nil {
				panic(err)
			}
			_, ok := etcdKeyPrefixesToCheck[string(kv.Key)]
			if ok {
				resourceToHandle = true
			} else {
				parts := strings.Split(string(kv.Key), "/")
				if strings.Contains(string(kv.Key), "/kubernetes.io/deployments/") {
					//fmt.Println(len(parts))
				}
				//fmt.Println(etcdKeyPrefixesToCheck, string(kv.Key), len(parts))
				if len(parts) == 4 {
					// trattasi di non namespaced resource, es.: /kubernetes.io/namespaces/prinquest
					result := strings.Join(parts[:3], "/")
					_, ok := etcdKeyPrefixesToCheck[result]
					if ok {
						resourceToHandle = true
					}
				}
				if len(parts) == 5 {
					// trattasi di namespaced resource
					if koff.AllNamespaces && !strings.Contains(strings.Join(parts[2:3], "/"), ".") {
						result := strings.Join(parts[:3], "/")
						//if strings.Contains(string(kv.Key), "/kubernetes.io/deployments/") {
						//	fmt.Println(strings.Join(parts[:3], "/"))
						//}
						_, ok := etcdKeyPrefixesToCheck[result]
						if ok {
							resourceToHandle = true
						}
					} else if koff.AllNamespaces && strings.Contains(strings.Join(parts[2:3], "/"), ".") {
						result := strings.Join(parts[:4], "/")
						//if strings.Contains(string(kv.Key), "/kubernetes.io/deployments/") {
						//	fmt.Println(strings.Join(parts[:3], "/"))
						//}
						_, ok := etcdKeyPrefixesToCheck[result]
						if ok {
							resourceToHandle = true
						}
					} else {
						result := strings.Join(parts[:4], "/")
						//if strings.Contains(string(kv.Key), "/kubernetes.io/deployments/") {
						//	fmt.Println(strings.Join(parts[:4], "/"))
						//}
						_, ok := etcdKeyPrefixesToCheck[result]
						if ok {
							resourceToHandle = true
						}
					}
				}
				if len(parts) == 6 {
					// trattasi di namespaced resource:
					// /kubernetes.io/services/specs/prinquest/prinquest
					// /kubernetes.io/services/endpoints/prinquest/prinquest
					if koff.AllNamespaces {
						result := strings.Join(parts[:4], "/")
						_, ok := etcdKeyPrefixesToCheck[result]
						if ok {
							resourceToHandle = true
						}
					} else {
						result := strings.Join(parts[:5], "/")
						_, ok := etcdKeyPrefixesToCheck[result]
						if ok {
							resourceToHandle = true
						}
					}
				}
			}
			if resourceToHandle {
				koff.EtcdKubeKeysToGet = append(koff.EtcdKubeKeysToGet, string(kv.Key))
			}
			return nil
		})
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func handleKubeKey(koff *types.KoffCommand, kubeKey string) error {
	var outputx bytes.Buffer
	if err := koff.EtcdDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("cannot get hash of bucket %s", string("key"))
		}
		etcdKey, exists := koff.KubeKeysToEtcdKeys[kubeKey]
		if exists {
			//fmt.Println("etcd key exist for", kubeKey)
			etcdvalue := b.Get(etcdKey)
			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(etcdvalue); err != nil {
				panic(err)
			}

			//fmt.Println(kubeKey, string(kv.Key))
			////fmt.Println(string(kv.Key))
			unstruct := &unstructured.Unstructured{}
			err := unstruct.UnmarshalJSON(kv.Value)
			if err == nil {
				//fmt.Println("handling object, ", unstruct.GetKind(), unstruct.GetName())
				err := HandleObject(koff, *unstruct)
				if err != nil {
					panic(err)
				}
			} else {
				//fmt.Println("*******", string(kv.Key))
				_, err = etcd.DetectAndConvert(JsonMediaType, kv.Value, &outputx)
				if err != nil {
					////fmt.Println("errrorrorooror", string(kv.Key)) ### ERRRORE DA GESTIRE!
					return nil
				}
				err := unstruct.UnmarshalJSON(outputx.Bytes())
				if err == nil {
					//fmt.Println("handling object, ", unstruct.GetKind(), unstruct.GetName())
					err = HandleObject(koff, *unstruct)
					if err != nil {
						panic(err)
					}
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
