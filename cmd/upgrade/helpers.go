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
package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/gmeghnag/koff/vars"
	progressbar "github.com/schollz/progressbar/v3"
)

type Releases []Release
type Release map[string]any

func updateKoffExecutable(koffExecutablePath string, url string, desiredVersion string) (err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error: Expected response code 200 from request executed to " + url + ", received " + strconv.Itoa(resp.StatusCode))
	}
	defer resp.Body.Close()

	//bar := progressbar.Default(-1, "")
	bar := CustomBytes(desiredVersion,
		resp.ContentLength,
		"upgrading",
	)

	// Stream the download through the progress bar and install it atomically so
	// a failed or interrupted download never leaves the user without a working
	// binary.
	return installBinary(koffExecutablePath, io.TeeReader(resp.Body, bar))
}

// installBinary writes src to a temporary file next to dest and atomically
// renames it into place with executable permissions. If writing fails, dest is
// left untouched and the temporary file is removed.
func installBinary(dest string, src io.Reader) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".koff-download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup; on success the file has already been renamed away.
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dest)
}

func checkReleases(repoName string) error {
	resp, err := http.Get("https://api.github.com/repos/" + repoName + "/releases")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected response %d from GitHub releases API: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var koffReleases Releases
	if err := json.Unmarshal(body, &koffReleases); err != nil {
		return fmt.Errorf("could not parse GitHub releases response: %w", err)
	}

	if vars.KoffTag == "" {
		vars.KoffTag = "v0.9.1"
	}
	currentVer, err := parseTag(vars.KoffTag)
	if err != nil {
		return err
	}
	fmt.Println("koff version is " + vars.KoffTag)
	fmt.Println("")
	fmt.Println("Available updates:")
	fmt.Println("")
	for _, release := range koffReleases {
		tag, ok := release["tag_name"].(string)
		if !ok {
			continue
		}
		availableReleaseVer, err := parseTag(tag)
		if err != nil {
			// Skip releases whose tag is not a valid vX.Y.Z semver.
			continue
		}
		if currentVer.LessThan(*availableReleaseVer) {
			fmt.Println(tag)
		}
	}
	return nil
}

// parseTag parses a "vX.Y.Z" release tag into a semantic version, returning an
// error instead of panicking on malformed input.
func parseTag(tag string) (*semver.Version, error) {
	if len(tag) < 2 || tag[0] != 'v' {
		return nil, fmt.Errorf("invalid version tag %q: expected form vX.Y.Z", tag)
	}
	v, err := semver.NewVersion(tag[1:])
	if err != nil {
		return nil, fmt.Errorf("invalid version tag %q: %w", tag, err)
	}
	return v, nil
}

func CustomBytes(desiredVersion string, maxBytes int64, description ...string) *progressbar.ProgressBar {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	return progressbar.NewOptions64(
		maxBytes,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(35),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\rkoff upgraded to "+desiredVersion+"                                                                        \n")
		}),
		progressbar.OptionSpinnerType(14),
		//progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}
