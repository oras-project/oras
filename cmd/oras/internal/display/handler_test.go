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
	"os"
	"reflect"
	"testing"

	"oras.land/oras/internal/testutils"

	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewPushHandler(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	_, _, err := NewPushHandler(printer, option.Format{Type: option.FormatTypeText.Name}, os.Stdout, mockFetcher.Fetcher)
	if err != nil {
		t.Errorf("NewPushHandler() error = %v, want nil", err)
	}
}

func TestNewAttachHandler(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	_, _, err := NewAttachHandler(printer, option.Format{Type: option.FormatTypeText.Name}, os.Stdout, mockFetcher.Fetcher)
	if err != nil {
		t.Errorf("NewAttachHandler() error = %v, want nil", err)
	}
}

func TestNewPullHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	_, _, err := NewPullHandler(printer, option.Format{Type: option.FormatTypeText.Name}, "", os.Stdout)
	if err != nil {
		t.Errorf("NewPullHandler() error = %v, want nil", err)
	}
}

func TestNewCopyHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	copyHandler, copyMetadataHandler := NewCopyHandler(printer, os.Stdout, nil)
	if _, ok := copyHandler.(*status.TTYCopyHandler); !ok {
		t.Errorf("expected *status.TTYCopyHandler actual %v", reflect.TypeOf(copyHandler))
	}
	if _, ok := copyMetadataHandler.(*text.CopyHandler); !ok {
		t.Errorf("expected metadata.CopyHandler actual %v", reflect.TypeOf(copyMetadataHandler))
	}
	copyHandler, copyMetadataHandler = NewCopyHandler(printer, nil, nil)
	if _, ok := copyHandler.(*status.TextCopyHandler); !ok {
		t.Errorf("expected *status.TextCopyHandler actual %v", reflect.TypeOf(copyHandler))
	}
	if _, ok := copyMetadataHandler.(*text.CopyHandler); !ok {
		t.Errorf("expected metadata.CopyHandler actual %v", reflect.TypeOf(copyMetadataHandler))
	}
}

func TestNewRepoTagsHandler(t *testing.T) {
	tests := []struct {
		name        string
		format      option.Format
		expectError bool
	}{
		{"text format", option.Format{Type: option.FormatTypeText.Name}, false},
		{"JSON format", option.Format{Type: option.FormatTypeJSON.Name}, false},
		{"Go template", option.Format{Type: option.FormatTypeGoTemplate.Name, Template: "{{.tags}}"}, false},
		{"unsupported", option.Format{Type: "unsupported"}, true},
	}

	// Test with stdout
	for _, tt := range tests {
		t.Run(tt.name+" with stdout", func(t *testing.T) {
			handler, err := NewRepoTagsHandler(os.Stdout, tt.format)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError {
				if err != nil {
					t.Errorf("error = %v, want nil", err)
				}
				if handler == nil {
					t.Error("returned nil handler")
				}
			}
		})
	}
}

func TestNewRepoListHandler(t *testing.T) {
	tests := []struct {
		name        string
		format      option.Format
		expectError bool
	}{
		{"text format", option.Format{Type: option.FormatTypeText.Name}, false},
		{"JSON format", option.Format{Type: option.FormatTypeJSON.Name}, false},
		{"Go template", option.Format{Type: option.FormatTypeGoTemplate.Name, Template: "{{.repositories}}"}, false},
		{"unsupported", option.Format{Type: "unsupported"}, true},
	}

	// Test with stdout
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := "example.com"
			namespace := "foo/bar"
			handler, err := NewRepoListHandler(os.Stdout, tt.format, registry, namespace)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError {
				if err != nil {
					t.Errorf("error = %v, want nil", err)
				}
				if handler == nil {
					t.Error("returned nil handler")
				}
			}
		})
	}
}

func TestNewBackupHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	repo := "test/repo"
	mockFetcher := testutils.NewMockFetcher()

	t.Run("with TTY", func(t *testing.T) {
		statusHandler, metadataHandler := NewBackupHandler(printer, os.Stdout, repo, mockFetcher.Fetcher)
		if _, ok := statusHandler.(*status.TTYBackupHandler); !ok {
			t.Errorf("expected *status.TTYBackupHandler actual %v", reflect.TypeOf(statusHandler))
		}
		if _, ok := metadataHandler.(*text.BackupHandler); !ok {
			t.Errorf("expected *text.BackupHandler actual %v", reflect.TypeOf(metadataHandler))
		}
	})

	t.Run("without TTY", func(t *testing.T) {
		statusHandler, metadataHandler := NewBackupHandler(printer, nil, repo, mockFetcher.Fetcher)
		if _, ok := statusHandler.(*status.TextBackupHandler); !ok {
			t.Errorf("expected *status.TextBackupHandler actual %v", reflect.TypeOf(statusHandler))
		}
		if _, ok := metadataHandler.(*text.BackupHandler); !ok {
			t.Errorf("expected *text.BackupHandler actual %v", reflect.TypeOf(metadataHandler))
		}
	})
}

func TestNewRestoreHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	mockFetcher := testutils.NewMockFetcher()

	t.Run("with TTY", func(t *testing.T) {
		statusHandler, metadataHandler := NewRestoreHandler(printer, os.Stdout, mockFetcher.Fetcher, false)
		if _, ok := statusHandler.(*status.TTYRestoreHandler); !ok {
			t.Errorf("expected *status.TTYRestoreHandler actual %v", reflect.TypeOf(statusHandler))
		}
		if _, ok := metadataHandler.(*text.RestoreHandler); !ok {
			t.Errorf("expected *text.RestoreHandler actual %v", reflect.TypeOf(metadataHandler))
		}
	})

	t.Run("without TTY", func(t *testing.T) {
		statusHandler, metadataHandler := NewRestoreHandler(printer, nil, mockFetcher.Fetcher, false)
		if _, ok := statusHandler.(*status.TextRestoreHandler); !ok {
			t.Errorf("expected *status.TextRestoreHandler actual %v", reflect.TypeOf(statusHandler))
		}
		if _, ok := metadataHandler.(*text.RestoreHandler); !ok {
			t.Errorf("expected *text.RestoreHandler actual %v", reflect.TypeOf(metadataHandler))
		}
	})
}
