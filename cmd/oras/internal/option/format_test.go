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
