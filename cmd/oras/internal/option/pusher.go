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
	ManifestExportPath      string
	PathValidationDisabled  bool
	ManifestAnnotations     string
	ManifestAnnotationSlice []string

	FileRefs []string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Pusher) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.ManifestExportPath, "export-manifest", "", "", "export the pushed manifest")
	fs.StringArrayVarP(&opts.ManifestAnnotationSlice, "annotation", "a", []string{}, "manifest annotations")
	fs.StringVarP(&opts.ManifestAnnotations, "manifest-annotations-file", "", "", "manifest annotation file")
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
func (opts *Pusher) LoadManifestAnnotations() (map[string]map[string]string, error) {
	var err error
	annotations := make(map[string]map[string]string)
	if opts.ManifestAnnotations != "" {
		if err = decodeJSON(opts.ManifestAnnotations, &annotations); err != nil {
			return nil, err
		}
	}
	if manifestAnnotationSliceLength := len(opts.ManifestAnnotationSlice); opts.ManifestAnnotations == "" &&
		manifestAnnotationSliceLength != 0 {
		if err = getAnnotationsMap(opts.ManifestAnnotationSlice, annotations); err != nil {
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

// getAnnotationsMap get resharp annotationslice to target type
func getAnnotationsMap(ManifestAnnotationSlice []string, annotations map[string]map[string]string) error {
	re := regexp.MustCompile(`=\s*`)
	rawAnnotationsMap := make(map[string]string)
	for _, rawAnnotation := range ManifestAnnotationSlice {
		rawAnnotation := re.Split(rawAnnotation, 2)
		rawAnnotation[0], rawAnnotation[1] = strings.TrimSpace(rawAnnotation[0]), strings.TrimSpace(rawAnnotation[1])
		if annotationLength := len(rawAnnotation); annotationLength != 2 {
			return errors.New("invalid annotation")
		}
		if _, ok := rawAnnotationsMap[rawAnnotation[0]]; ok {
			return errors.New("annotation key conflict")
		}
		rawAnnotationsMap[rawAnnotation[0]] = rawAnnotation[1]
	}
	annotations[annotationManifest] = rawAnnotationsMap
	return nil
}
