package template

import (
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func parseAndWrite(object any, templateStr string) error {
	t, err := template.New("format output").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		return err
	}
	return t.Execute(os.Stdout, object)
}
