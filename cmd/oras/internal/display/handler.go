package display

import (
	"os"

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
)

func NewPushHandler(format string, tty *os.File, verbose bool) (status.PushHandler, metadata.PushHandler) {
	var statusHandler status.PushHandler
	if tty != nil {
		statusHandler = status.NewTTYPushHandler(tty)
	} else if format == "" {
		statusHandler = status.NewTextPushHandler(verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.PushHandler
	switch format {
	case "":
		metadataHandler = text.NewPushHandler()
	case "json":
		metadataHandler = json.NewPushHandler()
	default:
		metadataHandler = template.NewPushHandler(format)
	}

	return statusHandler, metadataHandler
}

func NewAttachHandler(format string, tty *os.File, verbose bool) (status.AttachHandler, metadata.AttachHandler) {
	var statusHandler status.AttachHandler
	if tty != nil {
		statusHandler = status.NewTTYAttachHandler(tty)
	} else if format == "" {
		statusHandler = status.NewTextAttachHandler(verbose)
	} else {
		statusHandler = status.NewDiscardHandler()
	}

	var metadataHandler metadata.AttachHandler
	switch format {
	case "":
		metadataHandler = text.NewAttachHandler()
	case "json":
		metadataHandler = json.NewAttachHandler()
	default:
		metadataHandler = template.NewAttachHandler(format)
	}

	return statusHandler, metadataHandler
}
