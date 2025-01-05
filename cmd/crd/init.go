/*
Copyright © 2024 bverschueren <bverschueren@redhat.com>

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
package crd

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

//go:generate ../../hack/update-crd.sh
//go:generate mkdir -p _build
//go:generate cp -r ../../manifests ./_build

//go:embed _build/manifests
var staticCRD embed.FS

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize well known resources in the default directory",
	RunE: func(cmd *cobra.Command, args []string) error {

		// destination dir for the customresourcedefinition yamls
		crdOnDiskPath := userCrdPath()

		// list the statically built-in manifests
		staticCrdFiles, _ := staticCRD.ReadDir("_build/manifests")

		for _, crdPath := range staticCrdFiles {

			crdFile := "_build/manifests/" + crdPath.Name()

			raw, _ := staticCRD.ReadFile(crdFile)

			// try to parse it as a CRD
			obj := &apiextensionsv1.CustomResourceDefinition{}
			if err := yaml.Unmarshal(raw, obj); err != nil {
				klog.V(6).ErrorS(err, "failed to parse CustomResourceDefinition file", "crdFile", crdFile)
				continue
			}

			// if it's a valid CRD, write it to the predefined CRD location (default ~/.koff/customresourcedefinitions)
			onDiskDest := crdOnDiskPath + "/" + crdPath.Name()

			// do not overwrite existing (user-provided) CRD's
			if _, err := os.Stat(onDiskDest); errors.Is(err, os.ErrNotExist) {
				klog.V(3).Infof("Adding CRD yaml file for %s", obj.GetName())

				destination, err := os.Create(onDiskDest)
				if err != nil {
					klog.V(3).ErrorS(err, "failed to open destination CRD file", "onDiskDest", onDiskDest)
					continue
				}
				defer destination.Close()

				_, err = io.Copy(destination, bytes.NewReader(raw))
				if err != nil {
					klog.V(3).ErrorS(err, "failed to write CRD to destination file", "onDiskDest", onDiskDest)
				}
			}
		}
		return nil
	},
}

func userCrdPath() string {
	// TODO: move this to global helper function to prevent different defaults
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".koff", "customresourcedefinitions/")
}
