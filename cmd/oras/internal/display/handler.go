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

package display

import (
	"io"
	"os"

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
)

// NewPushHandler returns status and metadata handlers for push command.
func NewPushHandler(format string, tty *os.File, out io.Writer, verbose bool) (status.PushHandler, metadata.PushHandler) {
	var statusHandler status.PushHandler
	if tty != nil {
		statusHandler = status.NewTTYPushHandler(tty)
	} else if format == "" {
		statusHandler = status.NewTextPushHandler(out, verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PushHandler
	switch format {
	case "":
		metadataHandler = text.NewPushHandler(out)
	case "json":
		metadataHandler = json.NewPushHandler(out)
	default:
		metadataHandler = template.NewPushHandler(out, format)
	}

	return statusHandler, metadataHandler
}

// NewAttachHandler returns status and metadata handlers for attach command.
func NewAttachHandler(format string, tty *os.File, out io.Writer, verbose bool) (status.AttachHandler, metadata.AttachHandler) {
	var statusHandler status.AttachHandler
	if tty != nil {
		statusHandler = status.NewTTYAttachHandler(tty)
	} else if format == "" {
		statusHandler = status.NewTextAttachHandler(out, verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.AttachHandler
	switch format {
	case "":
		metadataHandler = text.NewAttachHandler(out)
	case "json":
		metadataHandler = json.NewAttachHandler(out)
	default:
		metadataHandler = template.NewAttachHandler(out, format)
	}

	return statusHandler, metadataHandler
}
