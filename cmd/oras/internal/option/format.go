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

	"github.com/spf13/pflag"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

const (
	// format types
	TypeJSON       = "json"
	TypeTree       = "tree"
	TypeTable      = "table"
	TypeGoTemplate = "go-template"
)

type FormatOption struct {
	Name  string
	Usage string
}

// Format is a flag to format metadata into output.
type Format struct {
	Type     string
	Template string
	input    string
	options  []FormatOption
}

// ApplyFlag implements FlagProvider.ApplyFlag.
func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	usage := "[Experimental] Format output using a custom template:"
	if len(opts.options) == 0 {
		opts.options = []FormatOption{
			{Name: TypeJSON, Usage: "Print in JSON format"},
			{Name: TypeGoTemplate, Usage: "Print output using the given Go template"},
		}
	}

	// generate usage string
	maxLength := 0
	for _, option := range opts.options {
		if len(option.Name) > maxLength {
			maxLength = len(option.Name)
		}
	}
	for _, option := range opts.options {
		usage += fmt.Sprintf("\n'%s':%s%s", option.Name, strings.Repeat(" ", maxLength-len(option.Name)+2), option.Usage)
	}
	usage += "."

	// apply flags
	fs.StringVar(&opts.input, "format", "", usage)
	fs.StringVar(&opts.Template, "template", "", `Template string used to format output`)
}

// Parse parses the input format flag.
func (opts *Format) Parse() error {
	if err := opts.parseFlag(); err != nil {
		return err
	}

	if opts.Template != "" && opts.Type != TypeGoTemplate {
		return fmt.Errorf("--template must be used with --format %s", TypeGoTemplate)
	}

	var optionalTypes []string
	for _, option := range opts.options {
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
		opts.Type = opts.input
		return nil
	}
	index := strings.Index(opts.input, "=")
	if index == -1 {
		// no proper template found in the type flag
		opts.Type = opts.input
		return nil
	} else if index == len(opts.Type)-1 || index == 0 {
		return fmt.Errorf("invalid format flag: %s", opts.input)
	}
	opts.Type = opts.Type[:index]
	opts.Template = opts.Type[index+1:]
	return nil
}

// SetFormatOptions sets the format options.
func (opts *Format) SetFormatOptions(options []FormatOption) {
	opts.options = options
}

// FormatError generate the error message for an invalid type.
func (opts *Format) TypeError() error {
	return fmt.Errorf("unsupported format type: %s", opts.Type)
}
