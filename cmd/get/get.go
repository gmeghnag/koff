package get

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gmeghnag/koff/pkg/deserializer"
	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/pkg/tablegenerator"
	"github.com/gmeghnag/koff/types"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cliprint "k8s.io/cli-runtime/pkg/printers"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

var Koff = types.NewKoffCommand()

//go:embed known-resources.yaml
var yamlData []byte

var GetCmd = &cobra.Command{
	Use: "get",
	RunE: func(cmd *cobra.Command, args []string) error {
		if Koff.OutputFormat == "wide" {
			Koff.Wide = true
		}
		koffConfigJson := types.Config{}
		var dataIn []byte
		if !terminal.IsTerminal(int(os.Stdin.Fd())) {
			infile := os.Stdin
			dataIn, _ = io.ReadAll(infile)
			Koff.FromInput = true
		} else {
			// gestire le eccezioni se il file non esiste ecc..
			file, _ := ioutil.ReadFile("/Users/gmeghnag/.koff/koff.json")
			_ = json.Unmarshal([]byte(file), &koffConfigJson)
			dataIn, _ = ioutil.ReadFile(koffConfigJson.InUse.Path)
		}
		err := helpers.ParseGetArgs(Koff, args, yamlData)
		if err != nil {
			klog.V(1).ErrorS(err, "ERROR")
			return err
		}
		if !Koff.FromInput && koffConfigJson.InUse.IsBundle {
			// TODO GESTISCI QUANDO è UN BUNDLE KOFF
			for resourceArg := range Koff.GetArgs {
				resourceType, resourceGroup, err := helpers.RetrieveKindGroup(resourceArg, yamlData)
				if err != nil {
					klog.V(1).ErrorS(err, "ERROR")
					return err
				}
				fmt.Println("+++", resourceType, resourceGroup)
			}
			//resourceType, resourceGroup, err := helpers.RetrieveKindGroup()
		} else if !Koff.FromInput && !koffConfigJson.InUse.IsBundle {
			// TODO GESTISCI QUANDO non è UN BUNDLE KOFF
			//for resourceArg := range Koff.GetArgs {
			//	resourceType, _, err := helpers.RetrieveKindGroup(resourceArg, yamlData)
			//	if err != nil {
			//		resourceType, _, err = helpers.RetrieveKindGroupFromCRDS(resourceArg, yamlData)
			//		if err != nil {
			//			fmt.Println("ERRORREEE")
			//		}
			//	}
			//}
			err = HandleDataIn(dataIn, Koff)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			//resourceType, resourceGroup, err := helpers.RetrieveKindGroup()
		} else {
			err = HandleDataIn(dataIn, Koff)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
		}

		err = KoffToStdOut(Koff)
		if err != nil {
			klog.V(2).ErrorS(err, "ERROR")
			return err
		}
		return nil
	},
}

func HandleDataIn(dataIn []byte, Koff *types.KoffCommand) error {
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
			err := HandleObject(Koff, unstructuredObject)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
		}
	} else {
		err := HandleObject(Koff, *unstructuredObject)
		if err != nil {
			return err
		}
	}
	return nil
}

func HandleObject(Koff *types.KoffCommand, obj unstructured.Unstructured) error {
	if Koff.FromInput && len(Koff.GetArgs) > 0 {
		resourcesNames, resourceTypePresent := Koff.GetArgs[strings.ToLower(obj.GetKind())]
		if !resourceTypePresent {
			resourcesNames, resourceTypePresent = Koff.GetArgs[strings.ToLower(obj.GetKind()+"."+strings.Split(obj.GetAPIVersion(), "/")[0])]
			if !resourceTypePresent {
				return nil
			}
		}
		Koff.ArgPresent[strings.ToLower(obj.GetKind())] = true
		_, resourceNamePresent := Koff.GetArgs[strings.ToLower(obj.GetKind())][obj.GetName()]
		if !resourceNamePresent {
			extendedResourceKind := obj.GetKind() + "." + strings.Split(obj.GetAPIVersion(), "/")[0]
			_, resourceNamePresent = Koff.GetArgs[strings.ToLower(extendedResourceKind)][obj.GetName()]
			if !resourceNamePresent && len(resourcesNames) > 0 {
				return nil
			}
		}
	}
	if Koff.Namespace != "" && obj.GetNamespace() != "" && Koff.Namespace != obj.GetNamespace() {
		return nil
	}
	Koff.LastKind = obj.GetKind()
	if Koff.OutputFormat == "yaml" || Koff.OutputFormat == "json" {
		if Koff.ShowManagedFields == false {
			obj.SetManagedFields(nil)
		}
		Koff.UnstructuredList.Items = append(Koff.UnstructuredList.Items, obj)
		return nil
	}
	if Koff.OutputFormat == "name" {
		if obj.GetAPIVersion() == "v1" {
			Koff.Output.WriteString(strings.ToLower(obj.GetKind()) + "/" + obj.GetName() + "\n")
		} else {
			Koff.Output.WriteString(strings.ToLower(obj.GetKind()) + "." + strings.Split(obj.GetAPIVersion(), "/")[0] + "/" + obj.GetName() + "\n")
		}
		return nil
	}
	rawObject, err := yaml.Marshal(obj.Object)
	if err != nil {
		klog.V(1).ErrorS(err, err.Error())
		return err
	}
	klog.V(3).Info("INFO deserializing ", obj.GetKind(), " ", obj.GetName())
	runtimeObjectType := deserializer.RawObjectToRuntimeObject(rawObject, Koff.Schema)
	if err := yaml.Unmarshal([]byte(rawObject), runtimeObjectType); err != nil {
		klog.V(3).Info(err, err.Error())
	}
	objectTable, err := tablegenerator.InternalResourceTable(Koff, runtimeObjectType, &obj)
	if err != nil {
		klog.V(3).Info("INFO ", fmt.Sprintf("%s: %s, %s", err.Error(), obj.GetKind(), obj.GetAPIVersion()))
		objectTable, err = tablegenerator.GenerateCustomResourceTable(Koff, obj)
		if err != nil {
			klog.V(1).ErrorS(err, err.Error())
			return err
		}
	}
	if Koff.CurrentKind == obj.GetObjectKind().GroupVersionKind().Kind {
		Koff.Table.Rows = append(Koff.Table.Rows, objectTable.Rows...)
	} else {
		// printo la tabella dell'oggetto precedente
		printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: Koff.NoHeaders, Wide: Koff.Wide, WithNamespace: false, ShowLabels: false})
		err = printer.PrintObj(&Koff.Table, &Koff.Output)
		if err != nil {
			klog.V(1).ErrorS(err, err.Error())
			return err
		}
		if Koff.CurrentKind != "" {
			Koff.Output.WriteByte('\n')
		}
		Koff.CurrentKind = obj.GetObjectKind().GroupVersionKind().Kind
		Koff.Table = metav1.Table{ColumnDefinitions: objectTable.ColumnDefinitions, Rows: objectTable.Rows}
	}
	return nil
}

func init() {
	GetCmd.Flags().BoolVarP(&Koff.ShowKind, "show-kind", "K", Koff.ShowKind, "Show kind.")
	GetCmd.Flags().BoolVar(&Koff.ShowManagedFields, "show-managed-fields", Koff.ShowManagedFields, "Show managedFields when output is one of: json, yaml.")
	GetCmd.Flags().BoolVarP(&Koff.ShowNamespace, "show-namespace", "N", Koff.ShowNamespace, "Show namespace.")
	GetCmd.Flags().BoolVar(&Koff.NoHeaders, "no-headers", Koff.NoHeaders, "Hide headers.")
	GetCmd.Flags().StringVarP(&Koff.OutputFormat, "output", "o", "", "Output format. One of: json|yaml|wide")
	GetCmd.Flags().StringVarP(&Koff.Namespace, "namespace", "n", "", "Namespace.")
}

func KoffToStdOut(*types.KoffCommand) error {
	if !Koff.FromInput && len(Koff.GetArgs) > 0 {
		for resource := range Koff.ArgPresent {
			exist, _ := Koff.ArgPresent[resource]
			if !exist {
				return fmt.Errorf(fmt.Sprintf("resource type \"%s\" not found.", resource))
			}
		}
	}
	printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: Koff.NoHeaders, Wide: Koff.Wide, WithNamespace: false, ShowLabels: false})
	if Koff.OutputFormat == "json" {
		if Koff.SingleResource && len(Koff.UnstructuredList.Items) == 1 {
			data, _ := json.MarshalIndent(Koff.UnstructuredList.Items[0].Object, "", "  ")
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
			data, _ := yaml.Marshal(Koff.UnstructuredList.Items[0].Object)
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
}
