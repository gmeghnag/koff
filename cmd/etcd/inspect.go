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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"sigs.k8s.io/yaml"
)

var output bytes.Buffer

const (
	StorageBinaryMediaType = "application/vnd.kubernetes.storagebinary"
	ProtobufMediaType      = "application/vnd.kubernetes.protobuf"
	YamlMediaType          = "application/yaml"
	JsonMediaType          = "application/json"
)

var (
	keyName, formatOutput string
)

var Inspect = &cobra.Command{

	Use:   "inspect",
	Short: "Inspect resources from etcd db (or snapshot) file.",
	Long:  "Select the etcd db file to inspect:\n\n  koff etcd inspect <filename> [<etcd_api_key>]",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(1)
		}
		inspectEtcd(args)
	},
}

func init() {
	Inspect.PersistentFlags().StringVarP(&formatOutput, "output", "o", "json", "Output format. One of: json|yaml")
}

func inspectEtcd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected at least one argument: etcd db file path.")
	}
	dbPath := args[0]

	if len(args) > 1 {
		keyName = args[1]
	}
	if _, err := os.Stat(dbPath); err != nil {
		return err
	}

	db, err := bolt.Open(dbPath, 0400, &bolt.Options{ReadOnly: false})
	if err != nil {
		fmt.Println("error trying to read", dbPath, "as boltdb file.")
		return err
	}
	defer db.Close()

	h := crc32.New(crc32.MakeTable(crc32.Castagnoli))

	if err = db.View(func(tx *bolt.Tx) error {
		// check snapshot file integrity first
		var dbErrStrings []string
		for dbErr := range tx.Check() {
			dbErrStrings = append(dbErrStrings, dbErr.Error())
		}
		if len(dbErrStrings) > 0 {
			return fmt.Errorf("snapshot file integrity check failed. %d errors found.\n"+strings.Join(dbErrStrings, "\n"), len(dbErrStrings))
		}
		c := tx.Cursor()
		//var out string
		for next, _ := c.First(); next != nil; next, _ = c.Next() {
			if string(next) == "key" {
				b := tx.Bucket(next)
				if b == nil {
					return fmt.Errorf("cannot get hash of bucket %s", string(next))
				}
				if _, err := h.Write(next); err != nil {
					return fmt.Errorf("cannot write bucket %s : %v", string(next), err)
				}

				err = b.ForEach(func(key, value []byte) error {
					var kv mvccpb.KeyValue
					if err := kv.Unmarshal(value); err != nil {
						panic(err)
					}
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
								_, err = DetectAndConvert(JsonMediaType, kv.Value, &output)
								err := unstruct.UnmarshalJSON(output.Bytes())
								if err == nil {
									data, _ := json.MarshalIndent(unstruct, "", "  ")
									data = append(data, '\n')
									fmt.Printf("%s", data)
									return nil
								}
							} else if formatOutput == "yaml" {
								_, err = DetectAndConvert(YamlMediaType, kv.Value, &output)
								fmt.Printf("%s", &output)
							}
							os.Exit(0)
						}
					} else {
						fmt.Println(string(kv.Key))
					}
					return nil
				})
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

type HelpType struct {
	Header map[string]string `json:"headers"`
	Kvs    []etcdObject      `json:"kvs"`
	Count  int               `json:"count"`
}

type etcdObject struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KeyValue struct {
	// key is the key in bytes. An empty key is not allowed.
	Key   []byte
	Value []byte
}

func keyDecoder(k, v []byte) {
	rev := bytesToRev(k)
	var kv mvccpb.KeyValue
	if err := kv.Unmarshal(v); err != nil {
		panic(err)
	}
	fmt.Printf("rev=%+v, value=[key %q | val %q | created %d | mod %d | ver %d]\n", rev, string(kv.Key), string(kv.Value), kv.CreateRevision, kv.ModRevision, kv.Version)
}

func bytesToRev(bytes []byte) revision {
	return revision{
		main: int64(binary.BigEndian.Uint64(bytes[0:8])),
		sub:  int64(binary.BigEndian.Uint64(bytes[9:])),
	}
}

type revision struct {
	main int64
	sub  int64
}

type R struct {
	ApiVersion string `json:"apiversion"`
}
