/*
Copyright The ORAS Authors.
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

package option

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/content"
)

const (
	annotationManifest = "$manifest"
)

// Pusher option struct.
type Pusher struct {
	ManifestExportPath     string
	PathValidationDisabled bool
	AnnotationsFilePath    string
	ManifestAnnotations    []string

	FileRefs []string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Pusher) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.ManifestExportPath, "export-manifest", "", "", "export the pushed manifest")
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil, "manifest annotations")
	fs.StringVarP(&opts.AnnotationsFilePath, "annotations-file", "", "", "path of the annotation file")
	fs.BoolVarP(&opts.PathValidationDisabled, "disable-path-validation", "", false, "skip path validation")
}

// ExportManifest saves the pushed manifest to a local file.
func (opts *Pusher) ExportManifest(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) error {
	if opts.ManifestExportPath == "" {
		return nil
	}
	manifestBytes, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return err
	}
	return os.WriteFile(opts.ManifestExportPath, manifestBytes, 0666)
}

// LoadManifestAnnotations loads the manifest annotation map.
func (opts *Pusher) LoadManifestAnnotations() (annotations map[string]map[string]string, err error) {
	annotations = make(map[string]map[string]string)
	if opts.AnnotationsFilePath != "" {
		if err = decodeJSON(opts.AnnotationsFilePath, &annotations); err != nil {
			return nil, err
		}
	}
	if annotationsLength := len(opts.ManifestAnnotations); annotationsLength != 0 {
		if opts.AnnotationsFilePath != "" {
			fmt.Fprintln(os.Stderr, "WARNING! --manifest--anotation are ignored if --manifest-annotation-file is specified.")
			return annotations, nil
		}
		if err = getAnnotationsMap(opts.ManifestAnnotations, annotations); err != nil {
			return nil, err
		}
	}
	return annotations, nil
}

// decodeJSON decodes a json file v to filename.
func decodeJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}

// getAnnotationsMap resharps annotationslice to k-v type and updates annotations
func getAnnotationsMap(ManifestAnnotations []string, annotations map[string]map[string]string) error {
	re := regexp.MustCompile(`=\s*`)
	rawAnnotationsMap := make(map[string]string)
	for _, rawAnnotationStr := range ManifestAnnotations {
		rawAnnotation := re.Split(rawAnnotationStr, 2)
		if annotationLength := len(rawAnnotation); annotationLength != 2 {
			return errors.New("annotation MUST be a key-value pair")
		}
		rawAnnotation[0], rawAnnotation[1] = strings.TrimSpace(rawAnnotation[0]), strings.TrimSpace(rawAnnotation[1])
		if _, ok := rawAnnotationsMap[rawAnnotation[0]]; ok {
			return errors.New("annotation key conflict")
		}
		rawAnnotationsMap[rawAnnotation[0]] = rawAnnotation[1]
	}
	annotations[annotationManifest] = rawAnnotationsMap
	return nil
}
