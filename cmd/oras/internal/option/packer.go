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
	"path/filepath"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/content"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/fileref"
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
	errPathValidation        = errors.New("absolute file path detected. If it's intentional, use --disable-path-validation flag to skip this check")
)

// Packer option struct.
type Packer struct {
	ManifestExportPath     string
	PathValidationDisabled bool
	AnnotationFilePath     string
	ManifestAnnotations    []string

	FileRefs []string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Packer) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.ManifestExportPath, "export-manifest", "", "", "`path` of the pushed manifest")
	fs.StringArrayVarP(&opts.ManifestAnnotations, "annotation", "a", nil, "manifest annotations")
	fs.StringVarP(&opts.AnnotationFilePath, "annotation-file", "", "", "path of the annotation file")
	fs.BoolVarP(&opts.PathValidationDisabled, "disable-path-validation", "", false, "skip path validation")
}

// ExportManifest saves the pushed manifest to a local file.
func (opts *Packer) ExportManifest(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) error {
	if opts.ManifestExportPath == "" {
		return nil
	}
	manifestBytes, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return err
	}
	return os.WriteFile(opts.ManifestExportPath, manifestBytes, 0666)
}
func (opts *Packer) Parse() error {
	if !opts.PathValidationDisabled {
		var failedPaths []string
		for _, path := range opts.FileRefs {
			// Remove the type if specified in the path <file>[:<type>] format
			path, _, err := fileref.Parse(path, "")
			if err != nil {
				return err
			}
			if filepath.IsAbs(path) {
				failedPaths = append(failedPaths, path)
			}
		}
		if len(failedPaths) > 0 {
			return fmt.Errorf("%w: %v", errPathValidation, strings.Join(failedPaths, ", "))
		}
	}
	return nil
}

// LoadManifestAnnotations loads the manifest annotation map.
func (opts *Packer) LoadManifestAnnotations() (annotations map[string]map[string]string, err error) {
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

// decodeJSON decodes a json file v to filename.
func decodeJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
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
