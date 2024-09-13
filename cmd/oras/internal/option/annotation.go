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
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

// Pre-defined annotation keys for annotation file
const (
	AnnotationManifest = "$manifest"
	AnnotationConfig   = "$config"
)

var (
	errAnnotationConflict    = errors.New("`--annotation` and `--annotation-file` cannot be both specified")
	errAnnotationFormat      = errors.New("annotation value doesn't match the required format")
	errAnnotationDuplication = errors.New("duplicate annotation key")
)

// Packer option struct.
type Annotation struct {
	AnnotationFilePath  string
	ManifestAnnotations []string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Annotation) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil, "manifest annotations")
	fs.StringVarP(&opts.AnnotationFilePath, "annotation-file", "", "", "path of the annotation file")
}

// LoadManifestAnnotations loads the manifest annotation map.
func (opts *Annotation) LoadManifestAnnotations() (annotations map[string]map[string]string, err error) {
	if opts.AnnotationFilePath != "" && len(opts.ManifestAnnotations) != 0 {
		return nil, errAnnotationConflict
	}
	if opts.AnnotationFilePath != "" {
		if err = decodeJSON(opts.AnnotationFilePath, &annotations); err != nil {
			return nil, &oerrors.Error{
				Err:            fmt.Errorf(`invalid annotation json file: failed to load annotations from %s`, opts.AnnotationFilePath),
				Recommendation: `Annotation file doesn't match the required format. Please refer to the document at https://oras.land/docs/how_to_guides/manifest_annotations`,
			}
		}
	}
	if len(opts.ManifestAnnotations) != 0 {
		annotations = make(map[string]map[string]string)
		if err = parseAnnotationFlags(opts.ManifestAnnotations, annotations); err != nil {
			return nil, err
		}
	}
	return
}

// parseAnnotationFlags parses annotation flags into a map.
func parseAnnotationFlags(flags []string, annotations map[string]map[string]string) error {
	manifestAnnotations := make(map[string]string)
	for _, anno := range flags {
		key, val, success := strings.Cut(anno, "=")
		if !success {
			return &oerrors.Error{
				Err:            errAnnotationFormat,
				Recommendation: `Please use the correct format in the flag: --annotation "key=value"`,
			}
		}
		if _, ok := manifestAnnotations[key]; ok {
			return fmt.Errorf("%w: %v, ", errAnnotationDuplication, key)
		}
		manifestAnnotations[key] = val
	}
	annotations[AnnotationManifest] = manifestAnnotations
	return nil
}
