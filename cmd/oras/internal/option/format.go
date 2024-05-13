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
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

// FormatType represents a custom description in help doc.
type FormatType struct {
	Name  string
	Usage string
}

// WithUsage returns a new format type with provided usage string.
func (ft *FormatType) WithUsage(usage string) *FormatType {
	return &FormatType{
		Name:  ft.Name,
		Usage: usage,
	}
}

// format types
var (
	FormatTypeJSON = &FormatType{
		Name:  "json",
		Usage: "Print in JSON format",
	}
	FormatTypeGoTemplate = &FormatType{
		Name:  "go-template",
		Usage: "Print in JSON format",
	}
	FormatTypeTable = &FormatType{
		Name:  "table",
		Usage: "Get direct referrers and output in table format",
	}
	FormatTypeTree = &FormatType{
		Name:  "tree",
		Usage: "Get referrers recursively and print in tree format",
	}
)

// Format contains input and parsed options for formatted output flags.
type Format struct {
	FormatFlag   string
	Type         string
	Template     string
	AllowedTypes []*FormatType
}

// ApplyFlag implements FlagProvider.ApplyFlag.
func (opts *Format) ApplyFlags(fs *pflag.FlagSet) {
	var buf bytes.Buffer
	_, _ = buf.WriteString("[Experimental] Format output using a custom template:")
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	if len(opts.AllowedTypes) == 0 {
		opts.AllowedTypes = []*FormatType{FormatTypeJSON, FormatTypeGoTemplate}
	}
	for _, t := range opts.AllowedTypes {
		_, _ = fmt.Fprintf(w, "\n'%s':\t%s", t.Name, t.Usage)
	}
	// apply flags
	fs.StringVar(&opts.FormatFlag, "format", opts.FormatFlag, buf.String())
	fs.StringVar(&opts.Template, "template", "", "[Experimental] Template string used to format output")
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

	if opts.Type == FormatTypeGoTemplate.Name && opts.Template == "" {
		return &oerrors.Error{
			Err:            fmt.Errorf("%q format specified but no template given", opts.Type),
			Recommendation: fmt.Sprintf("use `--format %q=TEMPLATE or --template TEMPLATE to specify the template", opts.Type),
		}
	}

	var optionalTypes []string
	for _, t := range opts.AllowedTypes {
		if opts.Type == t.Name {
			// type validation passed
			return nil
		}
		optionalTypes = append(optionalTypes, t.Name)
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
		if opts.Type != FormatTypeGoTemplate.Name {
			return fmt.Errorf("--template must be used with --format %s", FormatTypeGoTemplate.Name)
		}
		return nil
	}

	goTemplatePrefix := FormatTypeGoTemplate.Name + "="
	if strings.HasPrefix(opts.FormatFlag, goTemplatePrefix) {
		// add parameter to template
		opts.Type = FormatTypeGoTemplate.Name
		opts.Template = opts.FormatFlag[len(goTemplatePrefix):]
	}
	return nil
}
