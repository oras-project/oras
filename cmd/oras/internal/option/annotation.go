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
	errAnnotationTarget      = errors.New("annotation target must not be empty before the colon")
	errAnnotationLayerKey    = errors.New("annotation key must not be empty after the target")
)

// Annotation option struct.
type Annotation struct {
	// ManifestAnnotations contains raw input annotation flags.
	// Two formats are accepted:
	//   "key=value"             → manifest-level annotation
	//   "target:key=value"      → annotation scoped to an explicit target,
	//                             which is a filename (layer), "$config", or
	//                             "$manifest"
	ManifestAnnotations []string

	// Annotations contains parsed annotations keyed by target:
	// "$manifest" → manifest-level, "$config" → config-level, or a filename
	// → layer-level.
	Annotations map[string]map[string]string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Annotation) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil,
		`annotation in the format of "key=value"; prefix a target to scope it, e.g. "filename:key=value" (layer), "$config:key=value" (config), or "$manifest:key=value" (manifest)`)
}

// Parse parses the input annotation flags.
//
// Accepted formats:
//   - "key=value"           → manifest-level annotation
//   - "target:key=value"    → annotation scoped to an explicit target:
//     a filename (layer), "$config", or "$manifest"
//
// A colon before the first equals sign separates the target from the key.
// This matches the annotation-file JSON format where targets ("$manifest",
// "$config", or filenames) are top-level keys. OCI annotation keys use
// reverse-DNS with dots (not colons), so the colon is safe as a separator.
func (opts *Annotation) Parse(*cobra.Command) error {
	// The manifest target always has an entry so downstream packing can rely
	// on its presence even when only layer/config annotations are set.
	annotations := map[string]map[string]string{
		AnnotationManifest: {},
	}

	for _, anno := range opts.ManifestAnnotations {
		// Split on the first "=" to separate the key-part from the value.
		keyPart, value, ok := strings.Cut(anno, "=")
		if !ok {
			return &oerrors.Error{
				Err:            errAnnotationFormat,
				Recommendation: `Please use the correct format: --annotation "key=value" or --annotation "target:key=value"`,
			}
		}

		// A colon in the key-part (before the "=") scopes the annotation to
		// an explicit target. Without a colon it applies to the manifest.
		target, key := AnnotationManifest, keyPart
		if ref, refKey, scoped := strings.Cut(keyPart, ":"); scoped {
			if ref == "" {
				return &oerrors.Error{
					Err:            errAnnotationTarget,
					Recommendation: fmt.Sprintf(`Provide a target before the colon, e.g. --annotation "filename:%s=value", or drop it for a manifest annotation: --annotation "%s=value"`, refKey, refKey),
				}
			}
			if refKey == "" {
				return &oerrors.Error{
					Err:            errAnnotationLayerKey,
					Recommendation: fmt.Sprintf(`Provide an annotation key after the target: --annotation "%s:key=value"`, ref),
				}
			}
			target, key = ref, refKey
		}

		if annotations[target] == nil {
			annotations[target] = make(map[string]string)
		}
		if _, exists := annotations[target][key]; exists {
			return fmt.Errorf("%w: %q (target: %q)", errAnnotationDuplication, key, target)
		}
		annotations[target][key] = value
	}

	opts.Annotations = annotations
	return nil
}
