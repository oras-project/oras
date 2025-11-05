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

package root

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const docsDesc = `
Generate documentation files for ORAS.

This command can generate documentation for ORAS in the following formats:

- Markdown
- Man pages
- Bash autocompletions
`

type docsOptions struct {
	dest            string
	docTypeString   string
	topCmd          *cobra.Command
	generateHeaders bool
}

func docsCmd() *cobra.Command {
	o := &docsOptions{}

	cmd := &cobra.Command{
		Use:    "docs",
		Short:  "Generate documentation for ORAS",
		Long:   docsDesc,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.topCmd = cmd.Root()
			return o.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.dest, "dir", "./", "directory to which documentation is written")
	f.StringVar(&o.docTypeString, "type", "markdown", "the type of documentation to generate (markdown, man, bash)")
	f.BoolVar(&o.generateHeaders, "generate-headers", false, "generate standard headers for markdown files")

	_ = cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"bash", "man", "markdown"}, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

func (o *docsOptions) run() error {
	switch o.docTypeString {
	case "markdown", "mdown", "md":
		if o.generateHeaders {
			standardLinks := func(s string) string { return s }

			hdrFunc := func(filename string) string {
				base := filepath.Base(filename)
				name := strings.TrimSuffix(base, filepath.Ext(base))
				title := strings.ReplaceAll(name, "_", " ")
				// Capitalize first letter of each word
				words := strings.Fields(title)
				for i, word := range words {
					if len(word) > 0 {
						words[i] = strings.ToUpper(word[:1]) + word[1:]
					}
				}
				title = strings.Join(words, " ")
				return fmt.Sprintf("---\ntitle: \"%s\"\n---\n", title)
			}

			if err := doc.GenMarkdownTreeCustom(o.topCmd, o.dest, hdrFunc, standardLinks); err != nil {
				return err
			}
		} else {
			if err := doc.GenMarkdownTree(o.topCmd, o.dest); err != nil {
				return err
			}
		}
	case "man":
		header := &doc.GenManHeader{
			Title:   "ORAS",
			Section: "1",
		}
		if err := doc.GenManTree(o.topCmd, header, o.dest); err != nil {
			return err
		}
	case "bash":
		completionFile := filepath.Join(o.dest, "completions.bash")
		if err := o.topCmd.GenBashCompletionFile(completionFile); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown doc type %q. Try 'markdown' or 'man'", o.docTypeString)
	}

	log.Printf("Documentation successfully written to %s\n", o.dest)

	return nil
}
