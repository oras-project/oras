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

package trace

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// TextFormatter formats logs into text.
type TextFormatter struct{}

// logEntrySeperator is the separator between log entries.
const logEntrySeperator = "\n\n\n" // two empty lines

// Format renders a single log entry.
func (f *TextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buf bytes.Buffer

	timestamp := entry.Time.Format(time.RFC3339Nano)
	levelText := strings.ToUpper(entry.Level.String())
	buf.WriteString(fmt.Sprintf("[%s] %s: %s", timestamp, levelText, entry.Message))
	buf.WriteString(logEntrySeperator)

	// TODO: with fields?

	return buf.Bytes(), nil
}
