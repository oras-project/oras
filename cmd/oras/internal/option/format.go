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

// FormatType represents a format type.
type FormatType struct {
	// Name is the format type name.
	Name string
	// Usage is the usage string in help doc.
	Usage string
	// HasParams indicates whether the format type has parameters.
	HasParams bool
}

// WithUsage returns a new format type with provided usage string.
func (ft *FormatType) WithUsage(usage string) *FormatType {
	return &FormatType{
		Name:      ft.Name,
		HasParams: ft.HasParams,
		Usage:     usage,
	}
}

// format types
var (
	FormatTypeJSON = &FormatType{
		Name:  "json",
		Usage: "Print in JSON format",
	}
	FormatTypeGoTemplate = &FormatType{
		Name:      "go-template",
		Usage:     "Print output using the given Go template",
		HasParams: true,
	}
	// the table format is deprecated
	FormatTypeTable = &FormatType{
		Name:  "table",
		Usage: "[Deprecated] Get referrers and output in table format",
	}
	FormatTypeTree = &FormatType{
		Name:  "tree",
		Usage: "Get referrers and print in tree format",
	}
	FormatTypeText = &FormatType{
		Name:  "text",
		Usage: "Print in text format",
	}
)

// Format contains input and parsed options for formatted output flags.
type Format struct {
	FormatFlag   string
	Type         string
	Template     string
	allowedTypes []*FormatType
}

// SetTypes sets the default format type and allowed format types.
func (f *Format) SetTypes(defaultType *FormatType, otherTypes ...*FormatType) {
	f.FormatFlag = defaultType.Name
	f.allowedTypes = append(otherTypes, defaultType)
}

// ApplyFlags implements FlagProvider.ApplyFlag.
func (f *Format) ApplyFlags(fs *pflag.FlagSet) {
	buf := bytes.NewBufferString("[Experimental] format output using a custom template:")
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	for _, t := range f.allowedTypes {
		_, _ = fmt.Fprintf(w, "\n'%s':\t%s", t.Name, t.Usage)
	}
	_ = w.Flush()
	// apply flags
	fs.StringVar(&f.FormatFlag, "format", f.FormatFlag, buf.String())
	fs.StringVar(&f.Template, "template", "", "[Experimental] template string used to format output")
}

// Parse parses the input format flag.
func (f *Format) Parse(cmd *cobra.Command) error {
	// print deprecation message for table format
	if f.FormatFlag == FormatTypeTable.Name {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), "Format \"table\" is deprecated and will be removed in a future release.\n")
	}
	if err := f.parseFlag(); err != nil {
		return err
	}

	if f.Type == FormatTypeText.Name {
		// flag not specified
		return nil
	}

	if f.Type == FormatTypeGoTemplate.Name && f.Template == "" {
		return &oerrors.Error{
			Err:            fmt.Errorf("%q format specified but no template given", f.Type),
			Recommendation: fmt.Sprintf("use `--format %s=TEMPLATE` to specify the template", f.Type),
		}
	}

	var optionalTypes []string
	for _, t := range f.allowedTypes {
		if f.Type == t.Name {
			// type validation passed
			return nil
		}
		optionalTypes = append(optionalTypes, t.Name)
	}
	return &oerrors.Error{
		Err:            fmt.Errorf("invalid format type: %q", f.Type),
		Recommendation: fmt.Sprintf("supported types: %s", strings.Join(optionalTypes, ", ")),
	}
}

func (f *Format) parseFlag() error {
	f.Type = f.FormatFlag
	if f.Template != "" {
		// template explicitly set
		if f.Type != FormatTypeGoTemplate.Name {
			return fmt.Errorf("--template must be used with --format %s", FormatTypeGoTemplate.Name)
		}
		return nil
	}

	for _, t := range f.allowedTypes {
		if !t.HasParams {
			continue
		}
		prefix := t.Name + "="
		if strings.HasPrefix(f.FormatFlag, prefix) {
			// parse type and add parameter to template
			f.Type = t.Name
			f.Template = f.FormatFlag[len(prefix):]
		}
	}
	return nil
}
