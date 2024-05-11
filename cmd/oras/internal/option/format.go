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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const (
	// format types
	FormatTypeJSON       = "json"
	FormatTypeTree       = "tree"
	FormatTypeTable      = "table"
	FormatTypeGoTemplate = "go-template"
)

// FormatType represents a custom description in help doc.
type FormatType struct {
	Name  string
	Usage string
}

// Format contains input and parsed options for formatted output flags.
type Format struct {
	Type     string
	Template string
	// FormatFlag can be private once deprecated `--output` is removed from
	// `oras discover`
	FormatFlag string
	types      []FormatType
}

// ApplyFlag implements FlagProvider.ApplyFlag.
func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	usage := "[Experimental] Format output using a custom template:"
	if len(opts.types) == 0 {
		opts.types = []FormatType{
			{Name: FormatTypeJSON, Usage: "Print in JSON format"},
			{Name: FormatTypeGoTemplate, Usage: "Print output using the given Go template"},
		}
	}

	// generate usage string
	maxLength := 0
	for _, option := range opts.types {
		if len(option.Name) > maxLength {
			maxLength = len(option.Name)
		}
	}
	for _, option := range opts.types {
		usage += fmt.Sprintf("\n'%s':%s%s", option.Name, strings.Repeat(" ", maxLength-len(option.Name)+2), option.Usage)
	}

	// apply flags
	fs.StringVar(&opts.FormatFlag, "format", opts.FormatFlag, usage)
	fs.StringVar(&opts.Template, "template", "", "Template string used to format output")
}

// Parse parses the input format flag.
func (opts *Format) Parse(_ *cobra.Command) error {
	if err := opts.parseFlag(); err != nil {
		return err
	}

	if opts.Type == "" {
		// flag not specified
		return nil
	}

	var optionalTypes []string
	for _, option := range opts.types {
		if opts.Type == option.Name {
			// type validation passed
			return nil
		}
		optionalTypes = append(optionalTypes, option.Name)
	}
	return &oerrors.Error{
		Err:            fmt.Errorf("invalid format type: %q", opts.Type),
		Recommendation: fmt.Sprintf("supported types: %s", strings.Join(optionalTypes, ", ")),
	}
}

func (opts *Format) parseFlag() error {
	opts.Type = opts.FormatFlag
	if opts.Template != "" {
		// template explicitly set
		if opts.Type != FormatTypeGoTemplate {
			return fmt.Errorf("--template must be used with --format %s", FormatTypeGoTemplate)
		}
		return nil
	}

	goTemplatePrefix := FormatTypeGoTemplate + "="
	if strings.HasPrefix(opts.FormatFlag, goTemplatePrefix) {
		// add parameter to template
		opts.Type = FormatTypeGoTemplate
		opts.Template = opts.FormatFlag[len(goTemplatePrefix):]
	}
	return nil
}

// SetTypes resets the format options and default value.
func (opts *Format) SetTypes(types []FormatType) {
	opts.types = types
}

// SetTypesAndDefault resets the format options and default value.
// Caller should make sure that this function is used before applying flags.
func (opts *Format) SetTypesAndDefault(defaultType string, types []FormatType) {
	opts.FormatFlag = defaultType
	opts.types = types
}

// FormatError generates the error message for an invalid type.
func (opts *Format) TypeError() error {
	return fmt.Errorf("unsupported format type: %s", opts.Type)
}

// RawFormatFlag returns raw input of --format flag.
func (opts *Format) RawFormatFlag() string {
	return opts.FormatFlag
}
