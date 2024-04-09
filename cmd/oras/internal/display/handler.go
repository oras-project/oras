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
	"context"
	"errors"
	"io"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/table"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/metadata/tree"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
)

// ErrInvalidOutputType denotes the error for invalid output type.
var ErrInvalidOutputType = errors.New("output type can only be tree, table or json")

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

// NewPullHandler returns status and metadata handlers for pull command.
func NewPullHandler(format string, path string, tty *os.File, out io.Writer, verbose bool) (status.PullHandler, metadata.PullHandler) {
	var statusHandler status.PullHandler
	if tty != nil {
		statusHandler = status.NewTTYPullHandler(tty)
	} else if format == "" {
		statusHandler = status.NewTextPullHandler(out, verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PullHandler
	switch format {
	case "":
		metadataHandler = text.NewPullHandler(out)
	case "json":
		metadataHandler = json.NewPullHandler(out, path)
	default:
		metadataHandler = template.NewPullHandler(out, path, format)
	}
	return statusHandler, metadataHandler
}

// NewDiscoverHandler returns status and metadata handlers for discover command.
func NewDiscoverHandler(ctx context.Context, outputType string, path string, artifactType string, rawReference string, desc ocispec.Descriptor, repo option.ReadOnlyGraphTagFinderTarget, verbose bool) metadata.DiscoverHandler {
	if outputType == "tree" || outputType == "" {
		return tree.NewDiscoverHandler(ctx, path, repo, desc, artifactType, verbose)
	}
	switch outputType {
	case "table":
		return table.NewDiscoverHandler(ctx, outputType, path, artifactType, desc, repo, rawReference, verbose)
	case "json":
		return json.NewDiscoverHandler(ctx, outputType, path, artifactType, desc, repo)
	}
	return template.NewDiscoverHandler(ctx, outputType, path, artifactType, desc, repo)
}
