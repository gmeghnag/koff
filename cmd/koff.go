/*
Copyright © 2023 Koff Authors
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
package koff

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/gmeghnag/koff/cmd/etcd"
	"github.com/gmeghnag/koff/cmd/get"
	"github.com/gmeghnag/koff/cmd/upgrade"
	"github.com/gmeghnag/koff/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cliprint "k8s.io/cli-runtime/pkg/printers"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

var dataIn []byte
var Koff = types.NewKoffCommand()

var RootCmd = &cobra.Command{
	Use:           "koff",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if Koff.OutputFormat == "wide" {
			Koff.Wide = true
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			infile := os.Stdin
			dataIn, _ = io.ReadAll(infile)
			Koff.FromInput = true
		} else {
			return fmt.Errorf("expected kubernetes resource/s from piped input, not found")
		}
		unstructuredObject := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(dataIn), &unstructuredObject)
		if err != nil {
			klog.V(1).ErrorS(err, "ERROR")
			return err
		}
		if unstructuredObject.IsList() {
			unstructuredList := &unstructured.UnstructuredList{}
			err = yaml.Unmarshal([]byte(dataIn), &unstructuredList)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			for _, unstructuredObject := range unstructuredList.Items {
				err := get.HandleObject(Koff, unstructuredObject)
				if err != nil {
					klog.V(1).ErrorS(err, "ERROR")
					return err
				}
			}
		} else {
			err := get.HandleObject(Koff, *unstructuredObject)
			if err != nil {
				return err
			}
		}
		printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: Koff.NoHeaders, Wide: Koff.Wide, WithNamespace: false, ShowLabels: false})
		if Koff.OutputFormat == "json" {
			if Koff.SingleResource && len(Koff.UnstructuredList.Items) == 1 {
				data, _ := json.MarshalIndent(Koff.UnstructuredList.Items[0], "", "  ")
				data = append(data, '\n')
				fmt.Printf("%s", data)
				return nil
			} else {
				data, _ := json.MarshalIndent(Koff.UnstructuredList, "", "  ")
				data = append(data, '\n')
				fmt.Printf("%s", data)
				return nil
			}
		} else if Koff.OutputFormat == "yaml" {
			if Koff.SingleResource && len(Koff.UnstructuredList.Items) == 1 {
				data, _ := yaml.Marshal(Koff.UnstructuredList.Items[0])
				fmt.Printf("%s", data)
				return nil
			} else {
				data, _ := yaml.Marshal(Koff.UnstructuredList)
				fmt.Printf("%s", data)
				return nil
			}
		} else {
			if Koff.LastKind == Koff.CurrentKind {
				err := printer.PrintObj(&Koff.Table, &Koff.Output)
				if err != nil {
					klog.V(1).ErrorS(err, "ERROR")
					return err
				}
				Koff.Table = metav1.Table{}
			}
			Koff.Output.WriteTo(os.Stdout)
			return nil
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.Flags().BoolVarP(&Koff.ShowKind, "show-kind", "K", Koff.ShowKind, "Show kind.")
	RootCmd.Flags().BoolVar(&Koff.ShowManagedFields, "show-managed-fields", Koff.ShowManagedFields, "Show managedFields when output is one of: json, yaml, jsonpath.")
	RootCmd.Flags().BoolVarP(&Koff.ShowNamespace, "show-namespace", "N", Koff.ShowNamespace, "Show namespace.")
	RootCmd.Flags().BoolVar(&Koff.NoHeaders, "no-headers", Koff.NoHeaders, "Hide headers.")
	RootCmd.Flags().StringVarP(&Koff.OutputFormat, "output", "o", "", "Output format. One of: json|yaml|wide|jsonpath=...")
	RootCmd.Flags().StringVarP(&Koff.Namespace, "namespace", "n", "", "Namespace.")
	RootCmd.Flags().SortFlags = false
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlag(flag.CommandLine.Lookup("v"))
	RootCmd.AddCommand(
		get.GetCmd,
		UseCmd,
		upgrade.Upgrade,
		VersionCmd,
		etcd.EtcdCmd,
	)
}

func initConfig() {
	// create ~/.koff directory if it not exist
	user, err := user.Current()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// create ~/.koff/customresources directory if it not exist
	customResourcesPath := filepath.Join(user.HomeDir, ".koff", "customresourcedefinitions")
	if _, err := os.Stat(customResourcesPath); os.IsNotExist(err) {
		if err := os.MkdirAll(customResourcesPath, 0755); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

}
