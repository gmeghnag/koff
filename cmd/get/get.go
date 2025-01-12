package get

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/gmeghnag/koff/pkg/deserializer"
	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/pkg/tablegenerator"
	"github.com/gmeghnag/koff/types"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/ssh/terminal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cliprint "k8s.io/cli-runtime/pkg/printers"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

var Koff = types.NewKoffCommand()

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
			home, _ := os.UserHomeDir()
			file, _ := os.ReadFile(home + "/.koff/koff.json")
			_ = json.Unmarshal([]byte(file), &koffConfigJson)
			dataIn, _ = os.ReadFile(koffConfigJson.InUse.Path)
		}
		if !Koff.FromInput && koffConfigJson.InUse.IsEtcdDb {
			// QUANDO è UN DB ETCD
			Koff.IsEtcdDb = true
			var err error
			Koff.EtcdDb, err = bolt.Open(koffConfigJson.InUse.Path, 0400, &bolt.Options{ReadOnly: true})
			if err != nil {
				fmt.Println("error trying to read", koffConfigJson.InUse.Path, "as boltdb file.")
				return err
			}
			defer Koff.EtcdDb.Close()
			populateCRDsFromEtcd(Koff, koffConfigJson.InUse.Path, args[0])
			err = helpers.ParseGetArgs(Koff, args)
			//fmt.Println(Koff.EtcdAliasToCrdKubeKey)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			EtcdKeyPrefixesToCheck := make(map[string]bool)
			for resourType := range Koff.GetArgs {
				if len(Koff.GetArgs[resourType]) == 0 {
					x, err := helpers.EtcdPrefixFromAlias(Koff, resourType, "")
					if err != nil {
						return err
					}
					EtcdKeyPrefixesToCheck[x] = false
				} else {
					for resourName := range Koff.GetArgs[resourType] {
						x, err := helpers.EtcdPrefixFromAlias(Koff, resourType, resourName)
						if err != nil {
							return err
						}
						EtcdKeyPrefixesToCheck[x] = false
					}
				}
				//GetResourcesFor(Koff, resourType)
			}
			GetResourcesFromEtcd(Koff, EtcdKeyPrefixesToCheck)
			sort.Strings(Koff.EtcdKubeKeysToGet)
			for i, kubeKey := range Koff.EtcdKubeKeysToGet {
				//fmt.Println(Koff.EtcdKubeKeysToGet[i-1], kubeKey)
				// remove duplicated kubeKey (I don't know why but some keys are duplicated)
				if i == 0 || Koff.EtcdKubeKeysToGet[i-1] != kubeKey {
					handleKubeKey(Koff, kubeKey)
				}
			}
		}
		if !Koff.FromInput && koffConfigJson.InUse.IsBundle {
			// TODO GESTISCI QUANDO è UN BUNDLE KOFF
			fmt.Println("bundle koff")
			err := helpers.ParseGetArgs(Koff, args)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			for resourceArg := range Koff.GetArgs {
				resourceType, resourceGroup, err := helpers.RetrieveKindGroup(Koff, resourceArg)
				if err != nil {
					klog.V(1).ErrorS(err, "ERROR")
					return err
				}
				fmt.Println("+++", resourceType, resourceGroup)
			}
			//resourceType, resourceGroup, err := helpers.RetrieveKindGroup()
		} else if !Koff.FromInput && !koffConfigJson.InUse.IsBundle && !koffConfigJson.InUse.IsEtcdDb {
			err := helpers.ParseGetArgs(Koff, args)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			Koff.IsBundle = false
			err = HandleDataIn(dataIn, Koff)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
		} else if !koffConfigJson.InUse.IsBundle && !koffConfigJson.InUse.IsEtcdDb {
			err := helpers.ParseGetArgs(Koff, args)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			err = HandleDataIn(dataIn, Koff)
			if err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
		}

		err := KoffToStdOut(Koff)
		if err != nil {
			klog.V(2).ErrorS(err, "ERROR")
			return err
		}
		return nil
	},
}

func HandleDataIn(dataIn []byte, Koff *types.KoffCommand) error {
	unstructuredObject := &unstructured.Unstructured{}
	err := yaml.Unmarshal(dataIn, &unstructuredObject)
	if err != nil {
		klog.V(1).ErrorS(err, "ERROR")
		return err
	}
	if unstructuredObject.IsList() {
		unstructuredList := &unstructured.UnstructuredList{}
		err = yaml.Unmarshal(dataIn, &unstructuredList)
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
	Koff.ArgPresent[strings.ToLower(obj.GetKind())] = true
	if (Koff.FromInput || !Koff.IsBundle) && len(Koff.GetArgs) > 0 {
		resourcesNames, resourceTypePresent := Koff.GetArgs[strings.ToLower(obj.GetKind())]
		if !resourceTypePresent {
			_, resourceTypeWithGroupPresent := Koff.GetArgs[strings.ToLower(obj.GetKind()+"."+strings.Split(obj.GetAPIVersion(), "/")[0])]
			if !resourceTypeWithGroupPresent && !resourceTypePresent {
				return nil
			}
		}
		_, resourceNamePresent := Koff.GetArgs[strings.ToLower(obj.GetKind())][obj.GetName()]
		if !resourceNamePresent {
			extendedResourceKind := obj.GetKind() + "." + strings.Split(obj.GetAPIVersion(), "/")[0]
			_, extendedResourceNamePresent := Koff.GetArgs[strings.ToLower(extendedResourceKind)][obj.GetName()]
			if (!resourceNamePresent && !extendedResourceNamePresent) && len(resourcesNames) > 0 {
				return nil
			}
		}
	}
	if Koff.Namespace != "" && obj.GetNamespace() != "" && Koff.Namespace != obj.GetNamespace() {
		return nil
	}
	Koff.LastKind = obj.GetKind()
	if Koff.OutputFormat == "yaml" || Koff.OutputFormat == "json" {
		if !Koff.ShowManagedFields {
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
	var objectTable *metav1.Table
	_, ok := Koff.KnownResources[strings.ToLower(obj.GetKind())]
	if ok {
		runtimeObjectType := deserializer.RawObjectToRuntimeObject(rawObject, Koff.Schema)
		if err := yaml.Unmarshal([]byte(rawObject), runtimeObjectType); err != nil {
			klog.V(3).Info(err, err.Error())
		}
		objectTable, err = tablegenerator.InternalResourceTable(Koff, runtimeObjectType, &obj)
		if err != nil {
			klog.V(3).Info("INFO ", fmt.Sprintf("%s: %s, %s", err.Error(), obj.GetKind(), obj.GetAPIVersion()))
			klog.V(1).ErrorS(err, err.Error())
			return err
		}
	} else {
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
	GetCmd.Flags().BoolVarP(&Koff.AllNamespaces, "all-namespaces", "A", Koff.ShowNamespace, "Show resources across all namespaces.")
	GetCmd.Flags().BoolVar(&Koff.NoHeaders, "no-headers", Koff.NoHeaders, "Hide headers.")
	GetCmd.Flags().StringVarP(&Koff.OutputFormat, "output", "o", "", "Output format. One of: json|yaml|wide")
	GetCmd.Flags().StringVarP(&Koff.Namespace, "namespace", "n", "", "Namespace.")
}

func KoffToStdOut(*types.KoffCommand) error {
	//fmt.Println("Koff.GetArgs", Koff.GetArgs)
	//fmt.Println("Koff.ArgPresent", Koff.ArgPresent)
	if len(Koff.GetArgs) > 0 {
		for resource := range Koff.ArgPresent {
			exist, _ := Koff.ArgPresent[resource]
			if !exist {
				return fmt.Errorf("resource type or alias \"%s\" not known", resource)
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
		} else if !Koff.SingleResource && len(Koff.UnstructuredList.Items) > 0 {
			data, _ := json.MarshalIndent(Koff.UnstructuredList, "", "  ")
			data = append(data, '\n')
			fmt.Printf("%s", data)
			return nil
		} else {
			if Koff.Namespace != "" {
				fmt.Printf("No resources found in %s namespace.\n", Koff.Namespace)
			} else {
				fmt.Println("No resources found.")
			}
			return nil
		}
	} else if Koff.OutputFormat == "yaml" {
		if Koff.SingleResource && len(Koff.UnstructuredList.Items) == 1 {
			data, _ := yaml.Marshal(Koff.UnstructuredList.Items[0].Object)
			fmt.Printf("%s", data)
			return nil
		} else if len(Koff.UnstructuredList.Items) > 0 {
			data, _ := yaml.Marshal(Koff.UnstructuredList)
			fmt.Printf("%s", data)
			return nil
		} else {
			if Koff.Namespace != "" {
				fmt.Printf("No resources found in %s namespace.\n", Koff.Namespace)
			} else {
				fmt.Println("No resources found.")
			}
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
		if Koff.Output.Len() == 0 {
			if Koff.Namespace != "" {
				fmt.Printf("No resources found in %s namespace.\n", Koff.Namespace)
			} else {
				fmt.Println("No resources found.")
			}
		} else {
			Koff.Output.WriteTo(os.Stdout)
		}
		return nil
	}
}
