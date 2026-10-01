package option

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestFormatParseUnsupportedText(t *testing.T) {
	f := Format{}
	f.SetTypes(FormatTypeTree, FormatTypeTable, FormatTypeJSON, FormatTypeGoTemplate)
	f.FormatFlag = FormatTypeText.Name

	cmd := &cobra.Command{}
	err := f.Parse(cmd)

	if err == nil || !strings.Contains(err.Error(), `invalid format type: "text"`) {
		t.Fatalf("got %v, want invalid format type error for text", err)
	}
}

func TestFormatParseUnsupportedTable(t *testing.T) {
	f := Format{}
	f.SetTypes(FormatTypeText, FormatTypeJSON, FormatTypeGoTemplate)
	f.FormatFlag = FormatTypeTable.Name

	cmd := &cobra.Command{}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := f.Parse(cmd)

	if err == nil || !strings.Contains(err.Error(), `invalid format type: "table"`) {
		t.Fatalf("got %v, want invalid format type error for table", err)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("got stderr %q, want empty", got)
	}
}
