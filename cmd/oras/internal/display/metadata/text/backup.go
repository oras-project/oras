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

	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
	"oras.land/oras/cmd/oras/internal/output"
)

// BackupHandler handles text metadata output for backup events.
type BackupHandler struct {
	printer *output.Printer
	repo    string
}

// NewBackupHandler returns a new handler for backup events.
func NewBackupHandler(repo string, printer *output.Printer) metadata.BackupHandler {
	return &BackupHandler{
		repo:    repo,
		printer: printer,
	}
}

// OnBackupCompleted implements metadata.BackupHandler.
func (bh *BackupHandler) OnBackupCompleted(tagsCount int, path string) error {
	return bh.printer.Printf("Successfully backed up %d tag(s) from %s to %s\n", tagsCount, bh.repo, path)
}

// OnTarExported implements metadata.BackupHandler.
func (bh *BackupHandler) OnTarExported(path string, size int64) error {
	return bh.printer.Printf("Exported to %s (%s)\n", path, humanize.ToBytes(size))
}

// OnTarExporting implements metadata.BackupHandler.
func (bh *BackupHandler) OnTarExporting(path string) error {
	return bh.printer.Printf("Exporting to %s\n", path)
}

// OnArtifactPulled implements metadata.BackupHandler.
func (bh *BackupHandler) OnArtifactPulled(tag string, referrerCount int) error {
	return bh.printer.Printf("Pulled tag %s and %d referrer(s)\n", tag, referrerCount)
}

// OnTagsFound implements metadata.BackupHandler.
func (bh *BackupHandler) OnTagsFound(tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	return bh.printer.Printf("Found %d tag(s) in %s: %s\n", len(tags), bh.repo, strings.Join(tags, ", "))
}

// Render implements metadata.BackupHandler.
func (bh *BackupHandler) Render() error {
	return nil
}
