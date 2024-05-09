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

// FormatType represents a custom type for formatting.
type FormatType struct {
	Name  string
	Usage string
}

// Format contains input and parsed options to format output.
type Format struct {
	Type     string
	Template string
	Input    string
	types    []FormatType
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
	fs.StringVar(&opts.Input, "format", opts.Input, usage)
	fs.StringVar(&opts.Template, "template", "", `Template string used to format output`)
}

// Parse parses the input format flag.
func (opts *Format) Parse(_ *cobra.Command) error {
	if err := opts.parseFlag(); err != nil {
		return err
	}

	if opts.Template != "" && opts.Type != FormatTypeGoTemplate {
		return fmt.Errorf("--template must be used with --format %s", FormatTypeGoTemplate)
	}
	if opts.Type == "" {
		// flag not specified
		return nil
	}

	var optionalTypes []string
	for _, option := range opts.types {
		if opts.Type == option.Name {
			return nil
		}
		optionalTypes = append(optionalTypes, option.Name)
	}
	return &oerrors.Error{
		Err:            fmt.Errorf("invalid format type: %s", opts.Type),
		Recommendation: fmt.Sprintf("supported types: %s", strings.Join(optionalTypes, ", ")),
	}
}

func (opts *Format) parseFlag() error {
	if opts.Template != "" {
		// template explicitly set
		opts.Type = opts.Input
		return nil
	}
	index := strings.Index(opts.Input, "=")
	if index == -1 {
		// no proper template found in the type flag
		opts.Type = opts.Input
		return nil
	} else if index == len(opts.Input)-1 || index == 0 {
		return fmt.Errorf("invalid format flag: %s", opts.Input)
	}
	opts.Type = opts.Input[:index]
	opts.Template = opts.Input[index+1:]
	return nil
}

// SetTypes resets the format options and default value.
func (opts *Format) SetTypes(types []FormatType) {
	opts.types = types
}

// SetTypesAndDefault resets the format options and default value.
func (opts *Format) SetTypesAndDefault(defaultType string, types []FormatType) {
	opts.Input = defaultType
	opts.types = types
}

// FormatError generate the error message for an invalid type.
func (opts *Format) TypeError() error {
	return fmt.Errorf("unsupported format type: %s", opts.Type)
}
