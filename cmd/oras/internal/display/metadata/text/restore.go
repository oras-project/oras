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

package text

import (
	"strings"
	"time"

	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
	"oras.land/oras/cmd/oras/internal/output"
)

// RestoreHandler handles text metadata output for restore command.
type RestoreHandler struct {
	printer *output.Printer
	dryRun  bool
}

// NewRestoreHandler creates a new RestoreHandler.
func NewRestoreHandler(printer *output.Printer, dryRun bool) *RestoreHandler {
	return &RestoreHandler{
		printer: printer,
		dryRun:  dryRun,
	}
}

// OnTarLoaded implements metadata.RestoreHandler.
func (rh *RestoreHandler) OnTarLoaded(path string, size int64) error {
	return rh.printer.Printf("Loaded backup archive: %s (%s)\n", path, humanize.ToBytes(size))
}

// OnTagsFound implements metadata.RestoreHandler.
func (rh *RestoreHandler) OnTagsFound(tags []string) error {
	if len(tags) == 0 {
		return rh.printer.Printf("No tags found in the backup\n")
	}
	if len(tags) <= 5 {
		// print small number of tags in one line
		return rh.printer.Printf("Found %d tag(s) in the backup: %s\n", len(tags), strings.Join(tags, ", "))
	}
	// print large number of tags line by line
	if err := rh.printer.Printf("Found %d tag(s) in the backup:\n", len(tags)); err != nil {
		return err
	}
	for _, tag := range tags {
		if err := rh.printer.Println(tag); err != nil {
			return err
		}
	}
	return nil
}

// OnArtifactPushed implements metadata.RestoreHandler.
func (rh *RestoreHandler) OnArtifactPushed(tag string, referrerCount int) error {
	if rh.dryRun {
		return rh.printer.Printf("Dry run: would push tag %s with %d referrer(s)\n", tag, referrerCount)
	}
	return rh.printer.Printf("Pushed tag %s with %d referrer(s)\n", tag, referrerCount)
}

// OnRestoreCompleted implements metadata.RestoreHandler.
func (rh *RestoreHandler) OnRestoreCompleted(tagsCount int, repo string, duration time.Duration) error {
	if rh.dryRun {
		return rh.printer.Printf("Dry run complete: %d tag(s) would be restored to %q (no data pushed)\n", tagsCount, repo)
	}
	return rh.printer.Printf("Successfully restored %d tag(s) to %q in %s\n", tagsCount, repo, humanize.FormatDuration(duration))
}

// Render implements metadata.RestoreHandler.
func (rh *RestoreHandler) Render() error {
	return nil
}
