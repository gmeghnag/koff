package get

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gmeghnag/koff/pkg/deserializer"
	helpers "github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/pkg/tablegenerator"
	"github.com/gmeghnag/koff/types"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/term"
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
		// TODO: the pipe-vs-config decision relies on stdin being a TTY. In
		// non-interactive, non-TTY contexts (scripts, CI, cron) with no piped
		// data this blocks on io.ReadAll or ignores the configured snapshot.
		// Consider an explicit flag (e.g. --filename/-f) or detecting an empty,
		// non-pipe stdin instead of term.IsTerminal alone.
		if !term.IsTerminal(int(os.Stdin.Fd())) {
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
			if err := GetFromEtcd(Koff, args); err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
		}
		// File or piped input (anything that is not an etcd snapshot).
		if !koffConfigJson.InUse.IsEtcdDb {
			if err := helpers.ParseGetArgs(Koff, args); err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			if err := HandleDataIn(dataIn, Koff); err != nil {
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
	if len(bytes.TrimSpace(dataIn)) == 0 {
		// No input (empty file / empty pipe): nothing to render.
		return nil
	}
	unstructuredObject := &unstructured.Unstructured{}
	err := yaml.Unmarshal(dataIn, &unstructuredObject)
	if err != nil {
		klog.V(1).ErrorS(err, "ERROR")
		return err
	}
	if unstructuredObject == nil || unstructuredObject.Object == nil {
		// Input decoded to null/empty document.
		return nil
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
	if !passesFilters(Koff, obj) {
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
		Koff.Output.WriteString(nameOutput(obj))
		return nil
	}
	objectTable, err := buildObjectTable(Koff, obj)
	if err != nil {
		return err
	}
	return mergeObjectTable(Koff, obj, objectTable)
}

// passesFilters records that the kind was seen and reports whether obj should be
// rendered given the requested resource names and namespace. It mutates Koff
// (ArgPresent) and must be called serially.
func passesFilters(Koff *types.KoffCommand, obj unstructured.Unstructured) bool {
	Koff.ArgPresent[strings.ToLower(obj.GetKind())] = true
	if len(Koff.GetArgs) > 0 {
		resourcesNames, resourceTypePresent := Koff.GetArgs[strings.ToLower(obj.GetKind())]
		if !resourceTypePresent {
			_, resourceTypeWithGroupPresent := Koff.GetArgs[strings.ToLower(obj.GetKind()+"."+strings.Split(obj.GetAPIVersion(), "/")[0])]
			if !resourceTypeWithGroupPresent && !resourceTypePresent {
				return false
			}
		}
		_, resourceNamePresent := Koff.GetArgs[strings.ToLower(obj.GetKind())][obj.GetName()]
		if !resourceNamePresent {
			extendedResourceKind := obj.GetKind() + "." + strings.Split(obj.GetAPIVersion(), "/")[0]
			_, extendedResourceNamePresent := Koff.GetArgs[strings.ToLower(extendedResourceKind)][obj.GetName()]
			if (!resourceNamePresent && !extendedResourceNamePresent) && len(resourcesNames) > 0 {
				return false
			}
		}
	}
	if Koff.Namespace != "" && obj.GetNamespace() != "" && Koff.Namespace != obj.GetNamespace() {
		return false
	}
	return true
}

func nameOutput(obj unstructured.Unstructured) string {
	if obj.GetAPIVersion() == "v1" {
		return strings.ToLower(obj.GetKind()) + "/" + obj.GetName() + "\n"
	}
	return strings.ToLower(obj.GetKind()) + "." + strings.Split(obj.GetAPIVersion(), "/")[0] + "/" + obj.GetName() + "\n"
}

// buildObjectTable renders a single object's table. For custom resources it uses
// the stateful (serial) CRD resolution. Use buildObjectTableReadOnly for
// concurrent rendering.
func buildObjectTable(Koff *types.KoffCommand, obj unstructured.Unstructured) (*metav1.Table, error) {
	if _, known := Koff.KnownResources[strings.ToLower(obj.GetKind())]; known {
		return knownResourceTable(Koff, obj)
	}
	return tablegenerator.GenerateCustomResourceTable(Koff, obj)
}

// buildObjectTableReadOnly is a concurrency-safe equivalent of buildObjectTable:
// it never mutates Koff, resolving custom-resource CRDs read-only (etcd preloads
// them). Safe to call from multiple goroutines.
func buildObjectTableReadOnly(Koff *types.KoffCommand, obj unstructured.Unstructured) (*metav1.Table, error) {
	if _, known := Koff.KnownResources[strings.ToLower(obj.GetKind())]; known {
		return knownResourceTable(Koff, obj)
	}
	crd := tablegenerator.ResolveCRDReadOnly(Koff, obj)
	return tablegenerator.BuildCustomResourceTable(Koff, obj, crd)
}

// knownResourceTable builds the table for a built-in resource. It only reads
// Koff (Schema, config, TableGenerator) and is safe for concurrent use.
func knownResourceTable(Koff *types.KoffCommand, obj unstructured.Unstructured) (*metav1.Table, error) {
	rawObject, err := yaml.Marshal(obj.Object)
	if err != nil {
		klog.V(1).ErrorS(err, err.Error())
		return nil, err
	}
	runtimeObjectType := deserializer.RawObjectToRuntimeObject(rawObject, Koff.Schema)
	if err := yaml.Unmarshal(rawObject, runtimeObjectType); err != nil {
		klog.V(3).Info(err, err.Error())
	}
	table, err := tablegenerator.InternalResourceTable(Koff, runtimeObjectType, &obj)
	if err != nil {
		klog.V(3).Info("INFO ", fmt.Sprintf("%s: %s, %s", err.Error(), obj.GetKind(), obj.GetAPIVersion()))
		klog.V(1).ErrorS(err, err.Error())
		return nil, err
	}
	return table, nil
}

// mergeObjectTable appends an object's rendered rows to the running table,
// flushing the previous kind's table when the kind changes. It mutates Koff and
// must be called serially in output order.
func mergeObjectTable(Koff *types.KoffCommand, obj unstructured.Unstructured, objectTable *metav1.Table) error {
	if Koff.CurrentKind == obj.GetObjectKind().GroupVersionKind().Kind {
		Koff.Table.Rows = append(Koff.Table.Rows, objectTable.Rows...)
		return nil
	}
	printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: Koff.NoHeaders, Wide: Koff.Wide})
	if err := printer.PrintObj(&Koff.Table, &Koff.Output); err != nil {
		klog.V(1).ErrorS(err, err.Error())
		return err
	}
	if Koff.CurrentKind != "" {
		Koff.Output.WriteByte('\n')
	}
	Koff.CurrentKind = obj.GetObjectKind().GroupVersionKind().Kind
	Koff.Table = metav1.Table{ColumnDefinitions: objectTable.ColumnDefinitions, Rows: objectTable.Rows}
	return nil
}

func init() {
	GetCmd.Flags().BoolVarP(&Koff.ShowKind, "show-kind", "K", Koff.ShowKind, "Show kind.")
	GetCmd.Flags().BoolVar(&Koff.ShowManagedFields, "show-managed-fields", Koff.ShowManagedFields, "Show managedFields when output is one of: json, yaml.")
	GetCmd.Flags().BoolVarP(&Koff.ShowNamespace, "show-namespace", "N", Koff.ShowNamespace, "Show namespace.")
	GetCmd.Flags().BoolVarP(&Koff.AllNamespaces, "all-namespaces", "A", Koff.AllNamespaces, "Show resources across all namespaces.")
	GetCmd.Flags().BoolVar(&Koff.NoHeaders, "no-headers", Koff.NoHeaders, "Hide headers.")
	GetCmd.Flags().StringVarP(&Koff.OutputFormat, "output", "o", "", "Output format. One of: json|yaml|wide")
	GetCmd.Flags().StringVarP(&Koff.Namespace, "namespace", "n", "", "Namespace.")
}

// KoffToStdOut renders the accumulated results to os.Stdout.
func KoffToStdOut(koff *types.KoffCommand) error {
	return KoffToWriter(koff, os.Stdout)
}

// KoffToWriter renders the accumulated results to out. It honours the selected
// output format (json/yaml/table) and reports an error if a requested resource
// type was never seen.
func KoffToWriter(koff *types.KoffCommand, out io.Writer) error {
	if len(koff.GetArgs) > 0 {
		for resource, present := range koff.ArgPresent {
			if !present {
				return fmt.Errorf("resource type or alias \"%s\" not known", resource)
			}
		}
	}
	noResources := func() {
		if koff.Namespace != "" {
			fmt.Fprintf(out, "No resources found in %s namespace.\n", koff.Namespace)
		} else {
			fmt.Fprintln(out, "No resources found.")
		}
	}
	switch koff.OutputFormat {
	case "json":
		if koff.SingleResource && len(koff.UnstructuredList.Items) == 1 {
			data, _ := json.MarshalIndent(koff.UnstructuredList.Items[0].Object, "", "  ")
			fmt.Fprintf(out, "%s\n", data)
		} else if !koff.SingleResource && len(koff.UnstructuredList.Items) > 0 {
			data, _ := json.MarshalIndent(koff.UnstructuredList, "", "  ")
			fmt.Fprintf(out, "%s\n", data)
		} else {
			noResources()
		}
		return nil
	case "yaml":
		if koff.SingleResource && len(koff.UnstructuredList.Items) == 1 {
			data, _ := yaml.Marshal(koff.UnstructuredList.Items[0].Object)
			fmt.Fprintf(out, "%s", data)
		} else if len(koff.UnstructuredList.Items) > 0 {
			data, _ := yaml.Marshal(koff.UnstructuredList)
			fmt.Fprintf(out, "%s", data)
		} else {
			noResources()
		}
		return nil
	default:
		if koff.LastKind == koff.CurrentKind {
			printer := cliprint.NewTablePrinter(cliprint.PrintOptions{NoHeaders: koff.NoHeaders, Wide: koff.Wide})
			if err := printer.PrintObj(&koff.Table, &koff.Output); err != nil {
				klog.V(1).ErrorS(err, "ERROR")
				return err
			}
			koff.Table = metav1.Table{}
		}
		if koff.Output.Len() == 0 {
			noResources()
		} else {
			koff.Output.WriteTo(out)
		}
		return nil
	}
}
