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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gmeghnag/koff/cmd/etcd"
	"github.com/gmeghnag/koff/cmd/get"
	"github.com/gmeghnag/koff/cmd/upgrade"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
	klog "k8s.io/klog/v2"
)

var dataIn []byte

// Koff is shared with the get command so the whole CLI builds its scheme and
// table generator only once at startup.
var Koff = get.Koff

var RootCmd = &cobra.Command{
	Use:           "koff",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if Koff.OutputFormat == "wide" {
			Koff.Wide = true
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			dataIn, _ = io.ReadAll(os.Stdin)
			Koff.FromInput = true
		} else {
			return fmt.Errorf("expected kubernetes resource/s from piped input, not found")
		}
		if err := get.HandleDataIn(dataIn, Koff); err != nil {
			klog.V(1).ErrorS(err, "ERROR")
			return err
		}
		return get.KoffToWriter(Koff, os.Stdout)
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.Flags().BoolVarP(&Koff.ShowKind, "show-kind", "K", Koff.ShowKind, "Show kind.")
	RootCmd.Flags().BoolVar(&Koff.ShowManagedFields, "show-managed-fields", Koff.ShowManagedFields, "Show managedFields when output is one of: json, yaml, jsonpath.")
	RootCmd.Flags().BoolVarP(&Koff.ShowNamespace, "show-namespace", "N", Koff.ShowNamespace, "Show namespace.")
	RootCmd.Flags().BoolVar(&Koff.NoHeaders, "no-headers", Koff.NoHeaders, "Hide headers.")
	RootCmd.Flags().StringVarP(&Koff.OutputFormat, "output", "o", "", "Output format. One of: json|yaml|wide")
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
	// Resolve the home directory consistently with the rest of the tool
	// (os.UserHomeDir honours $HOME), then ensure ~/.koff/customresourcedefinitions exists.
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	customResourcesPath := filepath.Join(home, ".koff", "customresourcedefinitions")
	if err := os.MkdirAll(customResourcesPath, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
