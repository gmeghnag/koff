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
	"strconv"
	"strings"

	"github.com/gmeghnag/koff/pkg/helpers"
	"github.com/gmeghnag/koff/types"
	"github.com/gmeghnag/koff/vars"

	"github.com/spf13/cobra"
)

var isKoffBundle, isEtcDb bool

func useContext(path string, koffConfigFile string) {
	//if path != "" {
	//	if !filepath.IsAbs(path) {
	//		fmt.Println("error: \"" + path + "\" is not an absolute path.")
	//		os.Exit(1)
	//	}
	//}
	// read json koffConfigFile

	if strings.HasSuffix(path, ".db") {
		isEtcDb = true
	}
	isDir, _ := helpers.IsDirectory(path)
	if isDir {
		_path, err := findKoffBundleIn(path)
		if err == nil {
			isKoffBundle = true
		}
		l := strings.Split(_path, "/")
		path = strings.Join(l[0:(len(l)-1)], "/")
		path = strings.TrimSuffix(path, "/")
	}

	file, _ := os.ReadFile(koffConfigFile)
	koffConfigJson := types.Config{}
	_ = json.Unmarshal([]byte(file), &koffConfigJson)

	config := types.Config{}
	config.InUse = types.InUse{Path: path, Namespace: "", IsBundle: isKoffBundle, IsEtcdDb: isEtcDb}

	file, _ = json.MarshalIndent(config, "", " ")
	_ = os.WriteFile(koffConfigFile, file, 0644)

}

func findKoffBundleIn(path string) (string, error) {
	numDirs := 0
	dirName := ""
	retPath := strings.TrimSuffix(path, "/")
	var retErr error
	timeStampFound := false
	namespacesFolderFound := false
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if file.IsDir() {
			dirName = file.Name()
			numDirs = numDirs + 1
			if file.Name() == "namespaces" {
				namespacesFolderFound = true
			}
		}
		if !file.IsDir() && file.Name() == "timestamp" {
			timeStampFound = true
		}
	}
	if namespacesFolderFound {
		return retPath + "/", retErr
	}
	if timeStampFound && (numDirs > 1 || numDirs == 0) {
		return path, fmt.Errorf("expected one directory in path: \"%s\", found: %s", path, strconv.Itoa(numDirs))
	}
	if !timeStampFound && numDirs == 1 {
		retPath, retErr = findKoffBundleIn(path + "/" + dirName)
	}
	if !timeStampFound && !namespacesFolderFound {
		return path, fmt.Errorf("timestamp and namespace folder not found")
	}
	return retPath + "/", retErr
}

// useCmd represents the use command
var UseCmd = &cobra.Command{
	Use:   "use",
	Short: "Select the resource to inspect",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ""
		if len(args) > 1 || len(args) == 0 {
			return fmt.Errorf("expect one arguemnt, found: %v", len(args))

		} else {
			path = args[0]
			if strings.HasSuffix(path, "/") {
				path = strings.TrimRight(path, "/")
			}
			if strings.HasSuffix(path, "\\") {
				path = strings.TrimRight(path, "\\")
			}
			path, _ = filepath.Abs(path)
			fileInfo, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("file \"%s\" does not exist", path)
				} else {
					return fmt.Errorf("%s", err)
				}
			}
			if !fileInfo.Mode().IsRegular() {
				return fmt.Errorf("\"%s\" is not a regular file", path)
			}
		}
		useContext(path, "/Users/gmeghnag/.koff/koff.json")
		return nil
	},
}

func init() {
	UseCmd.Flags().StringVarP(&vars.Id, "id", "i", "", "Id string for the bundle to use. If two bundle has the same id the first one will be used.")
}
