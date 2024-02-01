package display

import (
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
)

func NewPushMetadataHandler(format string) metadata.PushHandler {
	switch format {
	case "":
		return text.NewPushHandler()
	case "json":
		return json.NewPushHandler()
	default:
		return template.NewPushHandler(format)
	}
}

func NewAttachMetadataHandler(format string) metadata.AttachHandler {
	switch format {
	case "":
		return text.NewAttachHandler()
	case "json":
		return json.NewAttachHandler()
	default:
		return template.NewAttachHandler(format)
	}
}
