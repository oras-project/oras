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

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"oras.land/oras/cmd/oras/internal/display/content"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/descriptor"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/table"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/metadata/tree"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// NewPushHandler returns status and metadata handlers for push command.
func NewPushHandler(printer *output.Printer, format option.Format, tty *os.File) (status.PushHandler, metadata.PushHandler, error) {
	var statusHandler status.PushHandler
	if tty != nil {
		statusHandler = status.NewTTYPushHandler(tty)
	} else if format.Type == option.FormatTypeText.Name {
		statusHandler = status.NewTextPushHandler(printer)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PushHandler
	switch format.Type {
	case option.FormatTypeText.Name:
		metadataHandler = text.NewPushHandler(printer)
	case option.FormatTypeJSON.Name:
		metadataHandler = json.NewPushHandler(printer)
	case option.FormatTypeGoTemplate.Name:
		metadataHandler = template.NewPushHandler(printer, format.Template)
	default:
		return nil, nil, errors.UnsupportedFormatTypeError(format.Type)
	}
	return statusHandler, metadataHandler, nil
}

// NewAttachHandler returns status and metadata handlers for attach command.
func NewAttachHandler(printer *output.Printer, format option.Format, tty *os.File) (status.AttachHandler, metadata.AttachHandler, error) {
	var statusHandler status.AttachHandler
	if tty != nil {
		statusHandler = status.NewTTYAttachHandler(tty)
	} else if format.Type == option.FormatTypeText.Name {
		statusHandler = status.NewTextAttachHandler(printer)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.AttachHandler
	switch format.Type {
	case option.FormatTypeText.Name:
		metadataHandler = text.NewAttachHandler(printer)
	case option.FormatTypeJSON.Name:
		metadataHandler = json.NewAttachHandler(printer)
	case option.FormatTypeGoTemplate.Name:
		metadataHandler = template.NewAttachHandler(printer, format.Template)
	default:
		return nil, nil, errors.UnsupportedFormatTypeError(format.Type)
	}
	return statusHandler, metadataHandler, nil
}

// NewPullHandler returns status and metadata handlers for pull command.
func NewPullHandler(printer *output.Printer, format option.Format, path string, tty *os.File) (status.PullHandler, metadata.PullHandler, error) {
	var statusHandler status.PullHandler
	if tty != nil {
		statusHandler = status.NewTTYPullHandler(tty)
	} else if format.Type == option.FormatTypeText.Name {
		statusHandler = status.NewTextPullHandler(printer)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PullHandler
	switch format.Type {
	case option.FormatTypeText.Name:
		metadataHandler = text.NewPullHandler(printer)
	case option.FormatTypeJSON.Name:
		metadataHandler = json.NewPullHandler(printer, path)
	case option.FormatTypeGoTemplate.Name:
		metadataHandler = template.NewPullHandler(printer, path, format.Template)
	default:
		return nil, nil, errors.UnsupportedFormatTypeError(format.Type)
	}
	return statusHandler, metadataHandler, nil
}

// NewDiscoverHandler returns status and metadata handlers for discover command.
func NewDiscoverHandler(out io.Writer, format option.Format, path string, rawReference string, desc ocispec.Descriptor, verbose bool) (metadata.DiscoverHandler, error) {
	var handler metadata.DiscoverHandler
	switch format.Type {
	case option.FormatTypeTree.Name:
		handler = tree.NewDiscoverHandler(out, path, desc, verbose)
	case option.FormatTypeTable.Name:
		handler = table.NewDiscoverHandler(out, rawReference, desc, verbose)
	case option.FormatTypeJSON.Name:
		handler = json.NewDiscoverHandler(out, desc, path)
	case option.FormatTypeGoTemplate.Name:
		handler = template.NewDiscoverHandler(out, desc, path, format.Template)
	default:
		return nil, errors.UnsupportedFormatTypeError(format.Type)
	}
	return handler, nil
}

// NewManifestFetchHandler returns a manifest fetch handler.
func NewManifestFetchHandler(out io.Writer, format option.Format, outputDescriptor, pretty bool, outputPath string) (metadata.ManifestFetchHandler, content.ManifestFetchHandler, error) {
	var metadataHandler metadata.ManifestFetchHandler
	var contentHandler content.ManifestFetchHandler

	switch format.Type {
	case option.FormatTypeText.Name:
		// raw
		if outputDescriptor {
			metadataHandler = descriptor.NewManifestFetchHandler(out, pretty)
		} else {
			metadataHandler = metadata.NewDiscardHandler()
		}
	case option.FormatTypeJSON.Name:
		// json
		metadataHandler = json.NewManifestFetchHandler(out)
		if outputPath == "" {
			contentHandler = content.NewDiscardHandler()
		}
	case option.FormatTypeGoTemplate.Name:
		// go template
		metadataHandler = template.NewManifestFetchHandler(out, format.Template)
		if outputPath == "" {
			contentHandler = content.NewDiscardHandler()
		}
	default:
		return nil, nil, errors.UnsupportedFormatTypeError(format.Type)
	}

	if contentHandler == nil {
		contentHandler = content.NewManifestFetchHandler(out, pretty, outputPath)
	}
	return metadataHandler, contentHandler, nil
}
