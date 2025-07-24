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
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"oras.land/oras/cmd/oras/internal/output"
)

func TestBackupHandler_OnBackupCompleted(t *testing.T) {
	repo := "testRepo"
	path := "testPath"
	tagsCount := 5
	tests := []struct {
		name    string
		out     io.Writer
		wantErr bool
		want    string
	}{
		{
			name:    "good path",
			out:     &bytes.Buffer{},
			wantErr: false,
			want:    fmt.Sprintf("Successfully backed up %d tag(s) from %s to %s\n", tagsCount, repo, path),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			bh := NewBackupHandler(repo, printer)
			if err := bh.OnBackupCompleted(tagsCount, path); (err != nil) != tt.wantErr {
				t.Errorf("OnBackupCompleted() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.want {
					t.Errorf("OnBackupCompleted() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBackupHandler_OnTarExported(t *testing.T) {
	path := "test.tar"
	tests := []struct {
		name    string
		out     io.Writer
		size    int64
		wantErr bool
		want    string
	}{
		{
			name:    "good path with 1KiB size",
			out:     &bytes.Buffer{},
			size:    int64(1024),
			wantErr: false,
			want:    fmt.Sprintf("Exported to %s (1 KB)\n", path),
		},
		{
			name:    "good path with 0 size",
			out:     &bytes.Buffer{},
			size:    0,
			wantErr: false,
			want:    fmt.Sprintf("Exported to %s (0  B)\n", path),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			bh := NewBackupHandler("any", printer)
			if err := bh.OnTarExported(path, tt.size); (err != nil) != tt.wantErr {
				t.Errorf("OnTarExported() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.want {
					t.Errorf("OnTarExported() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBackupHandler_OnTarExporting(t *testing.T) {
	path := "test.tar"
	tests := []struct {
		name    string
		out     io.Writer
		wantErr bool
		want    string
	}{
		{
			name:    "good path",
			out:     &bytes.Buffer{},
			wantErr: false,
			want:    fmt.Sprintf("Exporting to %s\n", path),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			bh := NewBackupHandler("any", printer)
			if err := bh.OnTarExporting(path); (err != nil) != tt.wantErr {
				t.Errorf("OnTarExporting() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.want {
					t.Errorf("OnTarExporting() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBackupHandler_OnArtifactPulled(t *testing.T) {
	tag := "latest"
	referrerCount := 3
	tests := []struct {
		name    string
		out     io.Writer
		wantErr bool
		want    string
	}{
		{
			name:    "good path",
			out:     &bytes.Buffer{},
			wantErr: false,
			want:    fmt.Sprintf("Backed up tag %s and %d referrer(s)\n", tag, referrerCount),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			bh := NewBackupHandler("any", printer)
			if err := bh.OnArtifactPulled(tag, referrerCount); (err != nil) != tt.wantErr {
				t.Errorf("OnArtifactPulled() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.want {
					t.Errorf("OnArtifactPulled() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBackupHandler_OnTagsFound(t *testing.T) {
	repo := "testRepo"
	tests := []struct {
		name    string
		tags    []string
		out     io.Writer
		wantErr bool
		want    string
	}{
		{
			name:    "good path with tags",
			tags:    []string{"t1", "t2"},
			out:     &bytes.Buffer{},
			wantErr: false,
			want:    fmt.Sprintf("Found 2 tag(s) in %s: t1, t2\n", repo),
		},
		{
			name:    "good path with no tags",
			tags:    []string{},
			out:     &bytes.Buffer{},
			wantErr: false,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := output.NewPrinter(tt.out, os.Stderr)
			bh := NewBackupHandler(repo, printer)
			if err := bh.OnTagsFound(tt.tags); (err != nil) != tt.wantErr {
				t.Errorf("OnTagsFound() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				got := tt.out.(*bytes.Buffer).String()
				if got != tt.want {
					t.Errorf("OnTagsFound() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestBackupHandler_Render(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "good path",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bh := &BackupHandler{}
			if err := bh.Render(); (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
