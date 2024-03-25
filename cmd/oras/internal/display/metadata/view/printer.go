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

package view

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"

	"github.com/Masterminds/sprig/v3"
)

// Printer prints.
type Printer interface {
	Printf(format string, a ...any) (n int, err error)
	Println(a ...any) (n int, err error)
	PrintJSON(object any) error
	ParseAndWrite(object any, templateStr string) error
}

type printer struct {
	out io.Writer
}

// NewPrinter creates a new printer based on out.
func NewPrinter(out io.Writer) Printer {
	return &printer{out: out}
}

// Printf writes the formatted string to the out.
func (p *printer) Printf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(p.out, format, a...)
}

// Println writes the string to the out.
func (p *printer) Println(a ...any) (n int, err error) {
	return fmt.Fprintln(p.out, a...)
}

// PrintJSON writes the object as JSON to the out.
func (p *printer) PrintJSON(object any) error {
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(object)
}

// ParseAndWrite parses the template string and writes to the out.
func (p *printer) ParseAndWrite(object any, templateStr string) error {
	t, err := template.New("format output").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		return err
	}
	return t.Execute(p.out, object)
}
