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
)

// Annotation option struct.
type Annotation struct {
	// ManifestAnnotations contains raw input of manifest annotation "key=value" pairs
	ManifestAnnotations []string

	// Annotations contains parsed manifest and config annotations
	Annotations map[string]map[string]string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Annotation) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil, "manifest annotations")
}

// Parse parses the input annotation flags.
func (opts *Annotation) Parse(*cobra.Command) error {
	manifestAnnotations := make(map[string]string)
	for _, anno := range opts.ManifestAnnotations {
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
	opts.Annotations = map[string]map[string]string{
		AnnotationManifest: manifestAnnotations,
	}
	return nil
}
