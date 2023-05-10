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
package cmd

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/gmeghnag/koff/pkg/deserializer"
	"github.com/gmeghnag/koff/pkg/tablegenerator"
	"github.com/gmeghnag/koff/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cliprint "k8s.io/cli-runtime/pkg/printers"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

var dataIn []byte
var Koff = types.NewKoffCommand()

var RootCmd = &cobra.Command{
	Use: "koff",
	RunE: func(cmd *cobra.Command, args []string) error {
		//printr := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: false, Wide: false, WithNamespace: false, ShowLabels: false})
		printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: Koff.NoHeaders, Wide: Koff.Wide, WithNamespace: false})
		if len(args) > 1 {
			yamFile := args[0]
			dataIn, _ = ioutil.ReadFile(yamFile)
		} else {
			infile := os.Stdin
			dataIn, _ = io.ReadAll(infile)
			Koff.FromInput = true
		}
		unstructuredObject := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(dataIn), &unstructuredObject)
		if err != nil {
			log.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		if unstructuredObject.IsList() {
			unstructuredList := &unstructured.UnstructuredList{}
			err = yaml.Unmarshal([]byte(dataIn), &unstructuredList)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			for _, unstructuredObject := range unstructuredList.Items {
				HandleObject(Koff, unstructuredObject)
			}
		} else {
			HandleObject(Koff, *unstructuredObject)
		}
		if Koff.LastKind == Koff.CurrentKind {
			//printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: false, Wide: false, WithNamespace: false})
			//err = printer.PrintObj(&Koff.Table, &Koff.Output)
			err = printer.PrintObj(&Koff.Table, &Koff.Output)
			if err != nil {
				log.Printf("Error: %s\n", err)
				os.Exit(1)
			}
			Koff.Table = metav1.Table{}
		}
		Koff.Output.WriteTo(os.Stdout)
		return nil
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.Flags().BoolVarP(&Koff.ShowKind, "show-kind", "K", Koff.ShowKind, "Show kind.")
	RootCmd.Flags().BoolVarP(&Koff.ShowNamespace, "show-namespace", "N", Koff.ShowNamespace, "Show namespace.")
	RootCmd.Flags().BoolVar(&Koff.Wide, "wide", Koff.Wide, "Show wide output.")
	RootCmd.Flags().BoolVar(&Koff.NoHeaders, "no-headers", Koff.NoHeaders, "Hide headers.")
	RootCmd.Flags().SortFlags = false
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlag(flag.CommandLine.Lookup("v"))
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

func HandleObject(Koff *types.KoffCommand, obj unstructured.Unstructured) {
	Koff.LastKind = obj.GetKind()
	rawObject, err := yaml.Marshal(obj.Object)
	if err != nil {
		log.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	// INIZIO
	// objectTable :=Koff.rawObjectToTable(rawObject, obj)
	runtimeObjectType := deserializer.RawObjectToRuntimeObject(rawObject, Koff.Schema)
	if err := yaml.Unmarshal([]byte(rawObject), runtimeObjectType); err != nil {
		//log.Printf(".... Error: %s\n", err)
	}
	objectTable, err := tablegenerator.InternalResourceTable(Koff, runtimeObjectType, &obj)
	if err != nil {
		// printer for the object is not registered or is a crd

		klog.V(3).Info("INFO ", fmt.Sprintf("%s: %s, %s", err.Error(), obj.GetKind(), obj.GetAPIVersion()))
		//log.printf fmt.Println(err, unstruct.GetKind(), unstruct.GetAPIVersion())
		objectTable, err = tablegenerator.GenerateCustomResourceTable(Koff, obj)
		if err != nil {
			klog.V(1).Info("ERROR ", fmt.Sprintf("%s", err.Error()))
			//objectTable = tablegenerator.UndefinedResourceTable(Koff, obj)

		}

	}
	// END
	// se l'oggetto è uguale a quello precedente
	// non printo newTable e non aggiungo ColumnDefinitions
	if Koff.CurrentKind == obj.GetObjectKind().GroupVersionKind().Kind {
		Koff.Table.Rows = append(Koff.Table.Rows, objectTable.Rows...)
	} else {
		// printo la tabella dell'oggetto precedente
		printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: Koff.NoHeaders, Wide: Koff.Wide, WithNamespace: false})
		err = printer.PrintObj(&Koff.Table, &Koff.Output)
		if err != nil {
			log.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		if Koff.CurrentKind != "" {
			Koff.Output.WriteByte('\n')
		}
		Koff.CurrentKind = obj.GetObjectKind().GroupVersionKind().Kind
		Koff.Table = metav1.Table{ColumnDefinitions: objectTable.ColumnDefinitions, Rows: objectTable.Rows}
	}
}
