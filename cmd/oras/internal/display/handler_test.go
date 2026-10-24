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

	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewPushHandler(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	printer := output.NewPrinter(os.Stdout, os.Stderr)

	tests := []struct {
		name           string
		format         option.Format
		tty            *os.File
		wantStatusType reflect.Type
		wantMetaType   reflect.Type
		wantError      bool
	}{
		{
			name:           "text",
			format:         option.Format{Type: option.FormatTypeText.Name},
			wantStatusType: reflect.TypeOf(&status.TextPushHandler{}),
			wantMetaType:   reflect.TypeOf(&text.PushHandler{}),
		},
		{
			name:           "text with TTY",
			format:         option.Format{Type: option.FormatTypeText.Name},
			tty:            os.Stdout,
			wantStatusType: reflect.TypeOf(&status.TTYPushHandler{}),
			wantMetaType:   reflect.TypeOf(&text.PushHandler{}),
		},
		{
			name:           "JSON",
			format:         option.Format{Type: option.FormatTypeJSON.Name},
			wantStatusType: reflect.TypeOf(status.DiscardHandler{}),
			wantMetaType:   reflect.TypeOf(&json.PushHandler{}),
		},
		{
			name:           "JSON with TTY",
			format:         option.Format{Type: option.FormatTypeJSON.Name},
			tty:            os.Stdout,
			wantStatusType: reflect.TypeOf(&status.TTYPushHandler{}),
			wantMetaType:   reflect.TypeOf(&json.PushHandler{}),
		},
		{
			name:           "Go template",
			format:         option.Format{Type: option.FormatTypeGoTemplate.Name, Template: "{{.name}}"},
			wantStatusType: reflect.TypeOf(status.DiscardHandler{}),
			wantMetaType:   reflect.TypeOf(&template.PushHandler{}),
		},
		{
			name:      "unsupported format",
			format:    option.Format{Type: "unsupported"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusHandler, metadataHandler, err := NewPushHandler(
				printer, tt.format, tt.tty, mockFetcher.Fetcher,
			)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if statusHandler != nil || metadataHandler != nil {
					t.Fatal("expected nil handlers for unsupported format")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewPushHandler() error = %v", err)
			}
			if got := reflect.TypeOf(statusHandler); got != tt.wantStatusType {
				t.Errorf("status handler type = %v, want %v", got, tt.wantStatusType)
			}
			if got := reflect.TypeOf(metadataHandler); got != tt.wantMetaType {
				t.Errorf("metadata handler type = %v, want %v", got, tt.wantMetaType)
			}
		})
	}
}

func TestNewAttachHandler(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	printer := output.NewPrinter(os.Stdout, os.Stderr)

	tests := []struct {
		name           string
		format         option.Format
		tty            *os.File
		wantStatusType reflect.Type
		wantMetaType   reflect.Type
		wantErr        bool
	}{
		{
			name:           "text",
			format:         option.Format{Type: option.FormatTypeText.Name},
			wantStatusType: reflect.TypeOf((*status.TextPushHandler)(nil)),
			wantMetaType:   reflect.TypeOf(text.NewAttachHandler(printer)),
		},
		{
			name:           "text with tty",
			format:         option.Format{Type: option.FormatTypeText.Name},
			tty:            os.Stdout,
			wantStatusType: reflect.TypeOf((*status.TTYPushHandler)(nil)),
			wantMetaType:   reflect.TypeOf(text.NewAttachHandler(printer)),
		},
		{
			name:           "json",
			format:         option.Format{Type: option.FormatTypeJSON.Name},
			wantStatusType: reflect.TypeOf(status.DiscardHandler{}),
			wantMetaType:   reflect.TypeOf(json.NewAttachHandler(printer)),
		},
		{
			name:           "template",
			format:         option.Format{Type: option.FormatTypeGoTemplate.Name},
			wantStatusType: reflect.TypeOf(status.DiscardHandler{}),
			wantMetaType:   reflect.TypeOf(template.NewAttachHandler(printer, "")),
		},
		{
			name:    "unsupported format",
			format:  option.Format{Type: "unsupported"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusHandler, metadataHandler, err := NewAttachHandler(
				printer, tt.format, tt.tty, mockFetcher.Fetcher,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for unsupported format")
				}
				if statusHandler != nil || metadataHandler != nil {
					t.Fatal("expected nil handlers for unsupported format")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewAttachHandler() error = %v", err)
			}
			if got := reflect.TypeOf(statusHandler); got != tt.wantStatusType {
				t.Errorf("status handler type = %v, want %v", got, tt.wantStatusType)
			}
			if got := reflect.TypeOf(metadataHandler); got != tt.wantMetaType {
				t.Errorf("metadata handler type = %v, want %v", got, tt.wantMetaType)
			}
		})
	}
}

func TestNewPullHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)

	tests := []struct {
		name           string
		format         option.Format
		tty            *os.File
		wantStatusType reflect.Type
		wantMetaType   reflect.Type
		wantErr        bool
	}{
		{
			name:           "text",
			format:         option.Format{Type: option.FormatTypeText.Name},
			wantStatusType: reflect.TypeOf((*status.TextPullHandler)(nil)),
			wantMetaType:   reflect.TypeOf(text.NewPullHandler(printer)),
		},
		{
			name:           "text with tty",
			format:         option.Format{Type: option.FormatTypeText.Name},
			tty:            os.Stdout,
			wantStatusType: reflect.TypeOf((*status.TTYPullHandler)(nil)),
			wantMetaType:   reflect.TypeOf(text.NewPullHandler(printer)),
		},
		{
			name:           "json",
			format:         option.Format{Type: option.FormatTypeJSON.Name},
			wantStatusType: reflect.TypeOf(status.DiscardHandler{}),
			wantMetaType:   reflect.TypeOf(json.NewPullHandler(printer, "")),
		},
		{
			name:           "template",
			format:         option.Format{Type: option.FormatTypeGoTemplate.Name},
			wantStatusType: reflect.TypeOf(status.DiscardHandler{}),
			wantMetaType:   reflect.TypeOf(template.NewPullHandler(printer, "", "")),
		},
		{
			name:    "unsupported format",
			format:  option.Format{Type: "unsupported"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusHandler, metadataHandler, err := NewPullHandler(
				printer, tt.format, "", tt.tty,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for unsupported format")
				}
				if statusHandler != nil || metadataHandler != nil {
					t.Fatal("expected nil handlers for unsupported format")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewPullHandler() error = %v", err)
			}
			if got := reflect.TypeOf(statusHandler); got != tt.wantStatusType {
				t.Errorf("status handler type = %v, want %v", got, tt.wantStatusType)
			}
			if got := reflect.TypeOf(metadataHandler); got != tt.wantMetaType {
				t.Errorf("metadata handler type = %v, want %v", got, tt.wantMetaType)
			}
		})
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
	out := os.Stdout

	tests := []struct {
		name     string
		format   option.Format
		wantType reflect.Type
		wantErr  bool
	}{
		{
			name:     "text",
			format:   option.Format{Type: option.FormatTypeText.Name},
			wantType: reflect.TypeOf(text.NewRepoTagsHandler(out)),
		},
		{
			name:     "json",
			format:   option.Format{Type: option.FormatTypeJSON.Name},
			wantType: reflect.TypeOf(json.NewRepoTagsHandler(out)),
		},
		{
			name:     "template",
			format:   option.Format{Type: option.FormatTypeGoTemplate.Name},
			wantType: reflect.TypeOf(template.NewRepoTagsHandler(out, "")),
		},
		{
			name:    "unsupported format",
			format:  option.Format{Type: "unsupported"},
			wantErr: true,
		},
	}

	// Test with stdout
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewRepoTagsHandler(out, tt.format)

			if tt.wantErr {
				if err == nil {
					t.Fatal("NewRepoTagsHandler() error = nil, want error")
				}
				if handler != nil {
					t.Fatal("NewRepoTagsHandler() handler != nil, want nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewRepoTagsHandler() error = %v, want nil", err)
			}
			if got := reflect.TypeOf(handler); got != tt.wantType {
				t.Errorf("handler type = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestNewRepoListHandler(t *testing.T) {
	out := os.Stdout
	registry := "registry.example.com"
	namespace := "example"

	tests := []struct {
		name     string
		format   option.Format
		wantType reflect.Type
		wantErr  bool
	}{
		{
			name:     "text",
			format:   option.Format{Type: option.FormatTypeText.Name},
			wantType: reflect.TypeOf(text.NewRepoListHandler(out, namespace)),
		},
		{
			name:     "json",
			format:   option.Format{Type: option.FormatTypeJSON.Name},
			wantType: reflect.TypeOf(json.NewRepoListHandler(out, registry)),
		},
		{
			name:     "template",
			format:   option.Format{Type: option.FormatTypeGoTemplate.Name},
			wantType: reflect.TypeOf(template.NewRepoListHandler(out, "", registry)),
		},
		{
			name:    "unsupported format",
			format:  option.Format{Type: "unsupported"},
			wantErr: true,
		},
	}

	// Test with stdout
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewRepoListHandler(out, tt.format, registry, namespace)

			if tt.wantErr {
				if err == nil {
					t.Fatal("NewRepoListHandler() error = nil, want error")
				}
				if handler != nil {
					t.Fatal("NewRepoListHandler() handler != nil, want nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewRepoListHandler() error = %v, want nil", err)
			}
			if got := reflect.TypeOf(handler); got != tt.wantType {
				t.Errorf("handler type = %v, want %v", got, tt.wantType)
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
