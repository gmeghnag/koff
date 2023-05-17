/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"fmt"

	"github.com/gmeghnag/koff/vars"

	"github.com/spf13/cobra"
)

// useCmd represents the use command
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print koff version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("koff version: %s\nhash: %s\nhttps://github.com/gmeghnag/koff\n", vars.KoffTag, vars.KoffHash)
		return nil
	},
}
