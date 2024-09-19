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

// Packer option struct.
type Annotation struct {
	ManifestAnnotationFlags []string                     // raw input of manifest annotation flags
	Annotations             map[string]map[string]string // parsed manifest and config annotations
}

// ApplyFlags applies flags to a command flag set.
func (opts *Annotation) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&opts.ManifestAnnotationFlags, "annotation", "a", nil, "manifest annotations")
}

func (opts *Annotation) Parse(*cobra.Command) error {
	opts.Annotations = make(map[string]map[string]string)
	manifestAnnotations := make(map[string]string)
	for _, anno := range opts.ManifestAnnotationFlags {
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
	opts.Annotations[AnnotationManifest] = manifestAnnotations
	return nil
}
