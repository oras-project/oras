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
	"oras.land/oras/cmd/oras/internal/option"
)

// NewPushHandler returns status and metadata handlers for push command.
func NewPushHandler(out io.Writer, format option.Format, tty *os.File, verbose bool) (status.PushHandler, metadata.PushHandler, error) {
	var statusHandler status.PushHandler
	if tty != nil {
		statusHandler = status.NewTTYPushHandler(tty)
	} else if format.Type == "" {
		statusHandler = status.NewTextPushHandler(out, verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PushHandler
	switch format.Type {
	case "":
		metadataHandler = text.NewPushHandler(out)
	case option.TypeJSON:
		metadataHandler = json.NewPushHandler(out)
	case option.TypeGoTemplate:
		metadataHandler = template.NewPushHandler(out, format.Template)
	default:
		return nil, nil, format.TypeError()
	}
	return statusHandler, metadataHandler, nil
}

// NewAttachHandler returns status and metadata handlers for attach command.
func NewAttachHandler(out io.Writer, format option.Format, tty *os.File, verbose bool) (status.AttachHandler, metadata.AttachHandler, error) {
	var statusHandler status.AttachHandler
	if tty != nil {
		statusHandler = status.NewTTYAttachHandler(tty)
	} else if format.Type == "" {
		statusHandler = status.NewTextAttachHandler(out, verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.AttachHandler
	switch format.Type {
	case "":
		metadataHandler = text.NewAttachHandler(out)
	case "json":
		metadataHandler = json.NewAttachHandler(out)
	case "go-template":
		metadataHandler = template.NewAttachHandler(out, format.Template)
	default:
		return nil, nil, format.TypeError()
	}
	return statusHandler, metadataHandler, nil
}

// NewPullHandler returns status and metadata handlers for pull command.
func NewPullHandler(out io.Writer, format option.Format, path string, tty *os.File, verbose bool) (status.PullHandler, metadata.PullHandler, error) {
	var statusHandler status.PullHandler
	if tty != nil {
		statusHandler = status.NewTTYPullHandler(tty)
	} else if format.Type == "" {
		statusHandler = status.NewTextPullHandler(out, verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PullHandler
	switch format.Type {
	case "":
		metadataHandler = text.NewPullHandler(out)
	case option.TypeJSON:
		metadataHandler = json.NewPullHandler(out, path)
	case option.TypeGoTemplate:
		metadataHandler = template.NewPullHandler(out, path, format.Template)
	default:
		return nil, nil, format.TypeError()
	}
	return statusHandler, metadataHandler, nil
}

// NewDiscoverHandler returns status and metadata handlers for discover command.
func NewDiscoverHandler(out io.Writer, format option.Format, path string, rawReference string, desc ocispec.Descriptor, verbose bool) (metadata.DiscoverHandler, error) {
	var handler metadata.DiscoverHandler
	switch format.Type {
	case option.TypeTree, "":
		handler = tree.NewDiscoverHandler(out, path, desc, verbose)
	case option.TypeTable:
		handler = table.NewDiscoverHandler(out, rawReference, desc, verbose)
	case option.TypeJSON:
		handler = json.NewDiscoverHandler(out, desc, path)
	case option.TypeGoTemplate:
		handler = template.NewDiscoverHandler(out, desc, path, format.Template)
	default:
		return nil, format.TypeError()
	}
	return handler, nil
}

// NewManifestFetchHandler returns a manifest fetch handler.
func NewManifestFetchHandler(out io.Writer, format option.Format, outputDescriptor, pretty bool, outputPath string) (metadata.ManifestFetchHandler, content.ManifestFetchHandler, error) {
	var metadataHandler metadata.ManifestFetchHandler
	var contentHandler content.ManifestFetchHandler

	switch format.Type {
	case "":
		// raw
		if outputDescriptor {
			metadataHandler = descriptor.NewManifestFetchHandler(out, pretty)
		} else {
			metadataHandler = metadata.NewDiscardHandler()
		}
	case option.TypeJSON:
		// json
		metadataHandler = json.NewManifestFetchHandler(out)
		if outputPath == "" {
			contentHandler = content.NewDiscardHandler()
		}
	case option.TypeGoTemplate:
		// go template
		metadataHandler = template.NewManifestFetchHandler(out, format.Template)
		if outputPath == "" {
			contentHandler = content.NewDiscardHandler()
		}
	default:
		return nil, nil, format.TypeError()
	}

	if contentHandler == nil {
		contentHandler = content.NewManifestFetchHandler(out, pretty, outputPath)
	}
	return metadataHandler, contentHandler, nil
}
