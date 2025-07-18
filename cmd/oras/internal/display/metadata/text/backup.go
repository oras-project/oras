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
	"oras.land/oras/cmd/oras/internal/output"
)

type BackupHandler struct {
	printer *output.Printer
	repo    string
	tags    []string
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
	bh.printer.Printf("Successfully backed up %d tag(s) from %q to %q\n", tagsCount, bh.repo, path)
	return nil
}

// OnExported implements metadata.BackupHandler.
func (bh *BackupHandler) OnExported(path string) error {
	// TODO: size?
	_ = bh.printer.Printf("Exported to %s\n", path)
	return nil
}

// OnExporting implements metadata.BackupHandler.
func (bh *BackupHandler) OnExporting(path string) error {
	_ = bh.printer.Printf("Exporting to %s\n", path)
	return nil
}

// OnArtifactPulled implements metadata.BackupHandler.
func (bh *BackupHandler) OnArtifactPulled(tag string, referrersCount int) error {
	_ = bh.printer.Printf("Pulled tag %q and %d referrer(s)\n", tag, referrersCount)
	return nil
}

// OnTagsFound implements metadata.BackupHandler.
func (bh *BackupHandler) OnTagsFound(tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	bh.tags = tags
	bh.printer.Printf("Found %d tag(s) in %q: %s\n", len(tags), bh.repo, strings.Join(tags, ", "))
	return nil
}

// Render implements metadata.BackupHandler.
func (bh *BackupHandler) Render() error {
	return nil
}
