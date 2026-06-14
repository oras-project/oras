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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

var (
	errAnnotationFormat      = errors.New("annotation value doesn't match the required format")
	errAnnotationDuplication = errors.New("duplicate annotation key")
	errAnnotationLayerKey    = errors.New("annotation key must not be empty after the file prefix")
)

// Annotation option struct.
type Annotation struct {
	// ManifestAnnotations contains raw input annotation flags.
	// Two formats are accepted:
	//   "key=value"               → manifest-level annotation
	//   "filename:key=value"      → layer-level annotation bound to filename
	ManifestAnnotations []string

	// Annotations contains parsed annotations keyed by target:
	// "$manifest" → manifest-level, or a filename → layer-level.
	Annotations map[string]map[string]string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Annotation) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil,
		`manifest annotations (e.g. "key=value") or layer annotations (e.g. "filename:key=value")`)
}

// Parse parses the input annotation flags.
//
// Accepted formats:
//   - "key=value"           → manifest-level annotation
//   - "filename:key=value"  → layer-level annotation bound to filename
//
// A colon before the first equals sign is treated as the file/key separator.
// This matches the annotation-file JSON format where filenames are top-level keys.
func (opts *Annotation) Parse(*cobra.Command) error {
	manifestAnnotations := make(map[string]string)
	fileAnnotations := make(map[string]map[string]string)

	for _, anno := range opts.ManifestAnnotations {
		// Split on the first "=" to separate key-part from value.
		keyPart, val, success := strings.Cut(anno, "=")
		if !success {
			return &oerrors.Error{
				Err:            errAnnotationFormat,
				Recommendation: `Please use the correct format: --annotation "key=value" or --annotation "filename:key=value"`,
			}
		}

		// A colon in keyPart (before the "=") signals a layer annotation.
		if fileRef, key, isLayer := strings.Cut(keyPart, ":"); isLayer {
			if key == "" {
				return &oerrors.Error{
					Err:            errAnnotationLayerKey,
					Recommendation: fmt.Sprintf(`Provide an annotation key after the file prefix: --annotation "%s:key=value"`, fileRef),
				}
			}
			if fileAnnotations[fileRef] == nil {
				fileAnnotations[fileRef] = make(map[string]string)
			}
			if _, ok := fileAnnotations[fileRef][key]; ok {
				return fmt.Errorf("%w: %v (layer: %v)", errAnnotationDuplication, key, fileRef)
			}
			fileAnnotations[fileRef][key] = val
		} else {
			// Manifest-level annotation.
			key := keyPart
			if _, ok := manifestAnnotations[key]; ok {
				return fmt.Errorf("%w: %v, ", errAnnotationDuplication, key)
			}
			manifestAnnotations[key] = val
		}
	}

	opts.Annotations = map[string]map[string]string{
		AnnotationManifest: manifestAnnotations,
	}
	for fileRef, annos := range fileAnnotations {
		opts.Annotations[fileRef] = annos
	}
	return nil
}
