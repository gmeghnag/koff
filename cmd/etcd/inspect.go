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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"sigs.k8s.io/yaml"
)

const (
	StorageBinaryMediaType = "application/vnd.kubernetes.storagebinary"
	ProtobufMediaType      = "application/vnd.kubernetes.protobuf"
	YamlMediaType          = "application/yaml"
	JsonMediaType          = "application/json"
)

var formatOutput string

var Inspect = &cobra.Command{

	Use:   "inspect",
	Short: "Inspect resources from etcd db (or snapshot) file.",
	Long:  "Select the etcd db file to inspect:\n\n  koff etcd inspect <filename> [<etcd_api_key>]",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(1)
		}
		return inspectEtcd(args, os.Stdout)
	},
}

// errStopIteration is a sentinel used to break out of bolt's ForEach once the
// requested key has been found; it is swallowed by the caller.
var errStopIteration = fmt.Errorf("stop iteration")

func init() {
	Inspect.PersistentFlags().StringVarP(&formatOutput, "output", "o", "json", "Output format. One of: json|yaml")
}

func inspectEtcd(args []string, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("expected at least one argument: etcd db file path")
	}
	dbPath := args[0]

	key := ""
	if len(args) > 1 {
		key = args[1]
	}
	if _, err := os.Stat(dbPath); err != nil {
		return err
	}

	// Snapshots must never be mutated: open the file strictly read-only.
	db, err := bolt.Open(dbPath, 0400, &bolt.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("error trying to read %q as boltdb file: %w", dbPath, err)
	}
	defer db.Close()

	return db.View(func(tx *bolt.Tx) error {
		// check snapshot file integrity first
		var dbErrStrings []string
		for dbErr := range tx.Check() {
			dbErrStrings = append(dbErrStrings, dbErr.Error())
		}
		if len(dbErrStrings) > 0 {
			return fmt.Errorf("snapshot file integrity check failed. %d errors found.\n"+strings.Join(dbErrStrings, "\n"), len(dbErrStrings))
		}
		b := tx.Bucket([]byte("key"))
		if b == nil {
			return fmt.Errorf("bucket %q not found in %q", "key", dbPath)
		}

		found := false
		iterErr := b.ForEach(func(_, value []byte) error {
			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(value); err != nil {
				return fmt.Errorf("unmarshalling etcd value: %w", err)
			}
			if key == "" {
				fmt.Fprintln(out, string(kv.Key))
				return nil
			}
			if string(kv.Key) != key {
				return nil
			}
			found = true
			if err := renderEtcdValue(kv.Value, formatOutput, out); err != nil {
				return err
			}
			return errStopIteration
		})
		if iterErr != nil && iterErr != errStopIteration {
			return iterErr
		}
		if key != "" && !found {
			return fmt.Errorf("key %q not found", key)
		}
		return nil
	})
}

// renderEtcdValue decodes a raw etcd value (either stored JSON, e.g. CRDs, or
// Kubernetes protobuf) into an unstructured object and writes it to out in the
// requested format ("json" or "yaml").
func renderEtcdValue(raw []byte, format string, out io.Writer) error {
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(raw); err == nil {
		return renderUnstructured(u, format, out)
	}
	var buf bytes.Buffer
	if _, err := DetectAndConvert(JsonMediaType, raw, &buf); err != nil {
		return err
	}
	if err := u.UnmarshalJSON(buf.Bytes()); err != nil {
		return err
	}
	return renderUnstructured(u, format, out)
}

func renderUnstructured(u *unstructured.Unstructured, format string, out io.Writer) error {
	switch format {
	case "yaml":
		data, err := yaml.Marshal(u)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "%s", data)
		return err
	default: // json
		data, err := json.MarshalIndent(u, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "%s\n", data)
		return err
	}
}
