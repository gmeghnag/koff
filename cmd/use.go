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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gmeghnag/koff/types"

	"github.com/spf13/cobra"
)

func useContext(path string, koffConfigFile string) error {
	config := types.Config{
		InUse: types.InUse{Path: path, IsEtcdDb: strings.HasSuffix(path, ".db")},
	}
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(koffConfigFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(koffConfigFile, data, 0644)
}

// useCmd represents the use command
var UseCmd = &cobra.Command{
	Use:   "use",
	Short: "Select the resource to inspect",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("expected exactly one argument, found: %v", len(args))
		}
		path := strings.TrimRight(args[0], "/\\")
		path, _ = filepath.Abs(path)
		fileInfo, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file \"%s\" does not exist", path)
			}
			return err
		}
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("\"%s\" is not a regular file", path)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return useContext(path, filepath.Join(home, ".koff", "koff.json"))
	},
}
