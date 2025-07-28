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

package root

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
)

func TestParseArtifactReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRepo string
		wantRefs []string
		wantErr  bool
	}{
		// Valid cases
		{
			name:     "valid reference with single tag",
			input:    "localhost:5000/repo:v1",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1"},
			wantErr:  false,
		},
		{
			name:     "valid reference with multiple tags",
			input:    "localhost:5000/repo:v1,v2,v3",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1", "v2", "v3"},
			wantErr:  false,
		},
		{
			name:     "complex repository path",
			input:    "localhost:5000/org/team/project:v1,v2",
			wantRepo: "localhost:5000/org/team/project",
			wantRefs: []string{"v1", "v2"},
			wantErr:  false,
		},
		{
			name:     "reference without tag",
			input:    "localhost:5000/repo",
			wantRepo: "localhost:5000/repo",
			wantRefs: nil,
			wantErr:  false,
		},
		{
			name:     "reference with empty tag",
			input:    "localhost:5000/repo:",
			wantRepo: "localhost:5000/repo",
			wantRefs: nil,
			wantErr:  false,
		},
		{
			name:     "valid tag with special characters",
			input:    "localhost:5000/repo:v1.0-beta_1",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1.0-beta_1"},
			wantErr:  false,
		},

		// Edge cases with empty tags
		{
			name:     "empty tag in middle of list",
			input:    "localhost:5000/repo:v1,,v2",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1", "v2"},
			wantErr:  false,
		},
		{
			name:     "empty first tag with valid second tag",
			input:    "localhost:5000/repo:,v1",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{"v1"},
			wantErr:  false,
		},
		{
			name:     "multiple empty tags",
			input:    "localhost:5000/repo:,,",
			wantRepo: "localhost:5000/repo",
			wantRefs: []string{},
			wantErr:  false,
		},

		// Error cases
		{
			name:     "empty reference",
			input:    "",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "digest reference not supported",
			input:    "localhost:5000/repo@sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "digest with additional tags",
			input:    "localhost:5000/repo@sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447,v1",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "invalid tag format with special chars",
			input:    "localhost:5000/repo:valid,invalid@tag",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "no repository and tag specified",
			input:    "localhost:5000",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "no repository specified",
			input:    "localhost:5000:v1",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "invalid repository format with space",
			input:    "localhost:5000/repo space:v1",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "tag exceeding max length",
			input:    "localhost:5000/repo:" + strings.Repeat("a", 129),
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "invalid tag starting with non-word character",
			input:    "localhost:5000/repo:.invalid",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
		{
			name:     "malformed reference with multiple colons",
			input:    "localhost:5000:abc/repo:v1",
			wantRepo: "",
			wantRefs: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, references, err := parseArtifactReferences(tt.input)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Validate results
			if repository != tt.wantRepo {
				t.Errorf("expected repository %q, got %q", tt.wantRepo, repository)
			}
			if !reflect.DeepEqual(references, tt.wantRefs) {
				t.Errorf("expected references %v, got %v", tt.wantRefs, references)
			}
		})
	}
}

func TestPrepareBackupOutput(t *testing.T) {
	// Create a temporary directory for our tests
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create a mock logger
	mockLogger := &mockLogger{}

	// Setup test context
	ctx := context.Background()

	t.Run("Directory output format", func(t *testing.T) {
		// Create an ingest directory to ensure it gets removed
		ingestDir := filepath.Join(tempDir, "ingest")
		if err := os.MkdirAll(ingestDir, 0755); err != nil {
			t.Fatalf("Failed to create ingest dir: %v", err)
		}

		mockHandler := &mockBackupHandler{}
		opts := &backupOptions{
			outputFormat: outputFormatDir,
			output:       filepath.Join(tempDir, "output-dir"),
		}

		err := prepareBackupOutput(ctx, tempDir, opts, mockLogger, mockHandler)
		if err != nil {
			t.Errorf("Expected no error for directory output, got: %v", err)
		}

		// Ensure ingest directory was removed
		if _, err := os.Stat(ingestDir); !os.IsNotExist(err) {
			t.Errorf("Expected ingest directory to be removed")
		}

		// Verify handler methods weren't called for directory output
		if mockHandler.tarExportingCalled {
			t.Errorf("OnTarExporting should not be called for directory output")
		}
		if mockHandler.tarExportedCalled {
			t.Errorf("OnTarExported should not be called for directory output")
		}
	})

	t.Run("Tar output format", func(t *testing.T) {
		// Create an ingest directory to ensure it gets removed
		ingestDir := filepath.Join(tempDir, "ingest")
		if err := os.MkdirAll(ingestDir, 0755); err != nil {
			t.Fatalf("Failed to create ingest dir: %v", err)
		}

		outputPath := filepath.Join(tempDir, "output.tar")
		opts := &backupOptions{
			outputFormat: outputFormatTar,
			output:       outputPath,
		}

		mockHandler := &mockBackupHandler{}
		err := prepareBackupOutput(ctx, tempDir, opts, mockLogger, mockHandler)
		if err != nil {
			t.Errorf("Expected no error for tar output, got: %v", err)
		}

		// Check if tar file was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Expected tar file to exist at %s", outputPath)
		}

		// Verify handler methods were called for tar output
		if !mockHandler.tarExportingCalled {
			t.Errorf("OnTarExporting wasn't called")
		}
		if !mockHandler.tarExportedCalled {
			t.Errorf("OnTarExported wasn't called")
		}

		// Clean up
		if err := os.Remove(outputPath); err != nil {
			t.Logf("Failed to remove output tar file: %v", err)
		}
	})

	t.Run("Error in OnTarExporting", func(t *testing.T) {
		ingestDir := filepath.Join(tempDir, "ingest")
		if err := os.MkdirAll(ingestDir, 0755); err != nil {
			t.Fatalf("Failed to create ingest dir: %v", err)
		}

		opts := &backupOptions{
			outputFormat: outputFormatTar,
			output:       filepath.Join(tempDir, "error.tar"),
		}

		expectedErr := fmt.Errorf("export error")
		mockHandler := &mockBackupHandler{
			tarExportingResult: expectedErr,
		}

		err := prepareBackupOutput(ctx, tempDir, opts, mockLogger, mockHandler)
		if err != expectedErr {
			t.Errorf("Expected error %v, got: %v", expectedErr, err)
		}
	})

	t.Run("Error in OnTarExported", func(t *testing.T) {
		ingestDir := filepath.Join(tempDir, "ingest")
		if err := os.MkdirAll(ingestDir, 0755); err != nil {
			t.Fatalf("Failed to create ingest dir: %v", err)
		}

		opts := &backupOptions{
			outputFormat: outputFormatTar,
			output:       filepath.Join(tempDir, "error.tar"),
		}

		expectedErr := fmt.Errorf("tar exported error")
		mockHandler := &mockBackupHandler{
			tarExportedResult: expectedErr,
		}

		err := prepareBackupOutput(ctx, tempDir, opts, mockLogger, mockHandler)
		if err != expectedErr {
			t.Errorf("Expected error %v, got: %v", expectedErr, err)
		}
	})

	t.Run("Non-existent output directory", func(t *testing.T) {
		// Create a temporary directory that we can control
		nonExistentDir := filepath.Join(tempDir, "non-existent")

		// Make sure it doesn't exist by removing it if it does
		_ = os.RemoveAll(nonExistentDir)

		// Setup a path in a non-existent directory that should trigger mkdir
		outputPath := filepath.Join(nonExistentDir, "nested", "output.tar")

		opts := &backupOptions{
			outputFormat: outputFormatTar,
			output:       outputPath,
		}

		mockHandler := &mockBackupHandler{}
		err := prepareBackupOutput(ctx, tempDir, opts, mockLogger, mockHandler)
		if err != nil {
			t.Errorf("Expected no error creating directories, got: %v", err)
		}

		// Verify the directory was created
		if _, err := os.Stat(filepath.Dir(outputPath)); os.IsNotExist(err) {
			t.Errorf("Expected output directory to be created")
		}
	})

	t.Run("Relative output path", func(t *testing.T) {
		// Save current directory
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current working directory: %v", err)
		}

		// Change to temp directory temporarily
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}
		defer func() {
			// Change back to original directory
			if err := os.Chdir(cwd); err != nil {
				t.Logf("Failed to restore working directory: %v", err)
			}
		}()

		// Use a relative path
		relPath := "./relative/output.tar"

		opts := &backupOptions{
			outputFormat: outputFormatTar,
			output:       relPath,
		}

		mockHandler := &mockBackupHandler{}
		err = prepareBackupOutput(ctx, tempDir, opts, mockLogger, mockHandler)
		if err != nil {
			t.Errorf("Expected no error with relative path, got: %v", err)
		}

		// Verify the file was created with the correct path
		absPath, _ := filepath.Abs(relPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Errorf("Expected tar file to exist at absolute path %s", absPath)
		}
	})
}

func Test_resolveTags(t *testing.T) {
	ctx := context.Background()
	repoName := "test/repo"

	// Mock descriptors
	desc1 := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:d5b7c742df27379894518554b73f7a3a03b4440ea435151a8b525a8d2555a0b2",
		Size:      123,
	}
	desc2 := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    "sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447",
		Size:      456,
	}

	// test server setup
	setupServer := func(handlers map[string]http.HandlerFunc) *httptest.Server {
		mux := http.NewServeMux()
		mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
			w.WriteHeader(http.StatusOK)
		})
		for path, handler := range handlers {
			mux.HandleFunc(path, handler)
		}
		return httptest.NewServer(mux)
	}

	// Common manifest handler
	manifestHandler := func(desc ocispec.Descriptor) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", desc.MediaType)
			w.Header().Set("Docker-Content-Digest", desc.Digest.String())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, desc.Size))
		}
	}

	t.Run("with specified tags", func(t *testing.T) {
		server := setupServer(map[string]http.HandlerFunc{
			fmt.Sprintf("/v2/%s/manifests/v1", repoName): manifestHandler(desc1),
			fmt.Sprintf("/v2/%s/manifests/v2", repoName): manifestHandler(desc2),
		})
		defer server.Close()

		repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/" + repoName)
		if err != nil {
			t.Fatalf("failed to create remote repository: %v", err)
		}
		repo.PlainHTTP = true

		tags, descs, err := resolveTags(ctx, repo, []string{"v1", "v2"})
		if err != nil {
			t.Fatalf("resolveTags() error = %v, wantErr nil", err)
		}
		if !reflect.DeepEqual(tags, []string{"v1", "v2"}) {
			t.Errorf("resolveTags() tags = %v, want %v", tags, []string{"v1", "v2"})
		}
		if len(descs) != 2 {
			t.Fatalf("resolveTags() expected 2 descriptors, got %d", len(descs))
		}
		if descs[0].Digest != desc1.Digest {
			t.Errorf("resolveTags() desc[0] digest = %v, want %v", descs[0].Digest, desc1.Digest)
		}
		if descs[1].Digest != desc2.Digest {
			t.Errorf("resolveTags() desc[1] digest = %v, want %v", descs[1].Digest, desc2.Digest)
		}
	})

	t.Run("error resolving specified tag", func(t *testing.T) {
		server := setupServer(map[string]http.HandlerFunc{
			fmt.Sprintf("/v2/%s/manifests/non-existent", repoName): func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
		})
		defer server.Close()

		repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/" + repoName)
		if err != nil {
			t.Fatalf("failed to create remote repository: %v", err)
		}
		repo.PlainHTTP = true

		_, _, err = resolveTags(ctx, repo, []string{"non-existent"})
		if wantErr := errdef.ErrNotFound; !errors.Is(err, wantErr) {
			t.Errorf("resolveTags() error = %v, wantErr %v", err, wantErr)
		}
	})

	t.Run("fetching all tags from repository", func(t *testing.T) {
		server := setupServer(map[string]http.HandlerFunc{
			fmt.Sprintf("/v2/%s/tags/list", repoName): func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"name":"` + repoName + `","tags":["v1","v2"]}`))
			},
			fmt.Sprintf("/v2/%s/manifests/v1", repoName): manifestHandler(desc1),
			fmt.Sprintf("/v2/%s/manifests/v2", repoName): manifestHandler(desc2),
		})
		defer server.Close()

		repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/" + repoName)
		if err != nil {
			t.Fatalf("failed to create remote repository: %v", err)
		}
		repo.PlainHTTP = true

		tags, descs, err := resolveTags(ctx, repo, nil)
		if err != nil {
			t.Fatalf("resolveTags() error = %v, wantErr nil", err)
		}
		expectedTags := []string{"v1", "v2"}
		if !reflect.DeepEqual(tags, expectedTags) {
			t.Errorf("resolveTags() tags = %v, want %v", tags, expectedTags)
		}
		if len(descs) != 2 {
			t.Fatalf("resolveTags() expected 2 descriptors, got %d", len(descs))
		}
		if descs[0].Digest != desc1.Digest {
			t.Errorf("resolveTags() desc[0] digest = %v, want %v", descs[0].Digest, desc1.Digest)
		}
		if descs[1].Digest != desc2.Digest {
			t.Errorf("resolveTags() desc[1] digest = %v, want %v", descs[1].Digest, desc2.Digest)
		}
	})

	t.Run("error listing tags from repository", func(t *testing.T) {
		server := setupServer(map[string]http.HandlerFunc{
			fmt.Sprintf("/v2/%s/tags/list", repoName): func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		})
		defer server.Close()

		repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/" + repoName)
		if err != nil {
			t.Fatalf("failed to create remote repository: %v", err)
		}
		repo.PlainHTTP = true

		_, _, err = resolveTags(ctx, repo, nil)
		if err == nil {
			t.Error("resolveTags() error = nil, wantErr not nil")
		}
	})

	t.Run("error resolving one of the listed tags", func(t *testing.T) {
		server := setupServer(map[string]http.HandlerFunc{
			fmt.Sprintf("/v2/%s/tags/list", repoName): func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"name":"` + repoName + `","tags":["v1","v2-bad"]}`))
			},
			fmt.Sprintf("/v2/%s/manifests/v1", repoName): manifestHandler(desc1),
			fmt.Sprintf("/v2/%s/manifests/v2-bad", repoName): func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
		})
		defer server.Close()

		repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/" + repoName)
		if err != nil {
			t.Fatalf("failed to create remote repository: %v", err)
		}
		repo.PlainHTTP = true

		_, _, err = resolveTags(ctx, repo, nil)
		if wantErr := errdef.ErrNotFound; !errors.Is(err, wantErr) {
			t.Errorf("resolveTags() error = %v, wantErr %v", err, wantErr)
		}
	})

	t.Run("empty tag list from repository", func(t *testing.T) {
		server := setupServer(map[string]http.HandlerFunc{
			fmt.Sprintf("/v2/%s/tags/list", repoName): func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"name":"` + repoName + `","tags":[]}`))
			},
		})
		defer server.Close()

		repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/" + repoName)
		if err != nil {
			t.Fatalf("failed to create remote repository: %v", err)
		}
		repo.PlainHTTP = true

		tags, descs, err := resolveTags(ctx, repo, nil)
		if err != nil {
			t.Fatalf("resolveTags() error = %v, wantErr nil", err)
		}
		if len(tags) != 0 {
			t.Errorf("resolveTags() tags = %v, want empty slice", tags)
		}
		if len(descs) != 0 {
			t.Errorf("resolveTags() descs = %v, want empty slice", descs)
		}
	})

	t.Run("target does not support tag listing", func(t *testing.T) {
		// Use a simple mock that doesn't implement registry.TagLister
		target := memory.New()
		_, _, err := resolveTags(ctx, target, nil)
		if wantErr := errTagListNotSupported; !errors.Is(err, wantErr) {
			t.Errorf("resolveTags() error = %v, wantErr %v", err, wantErr)
		}
	})
}

// Mock implementations
type mockLogger struct {
	debugMessages []string
}

func (m *mockLogger) WithField(key string, value interface{}) *logrus.Entry {
	return logrus.WithField(key, value)
}

func (m *mockLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	return logrus.WithFields(fields)
}

func (m *mockLogger) WithError(err error) *logrus.Entry {
	return logrus.WithError(err)
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	m.debugMessages = append(m.debugMessages, fmt.Sprintf(format, args...))
}

func (m *mockLogger) Infof(format string, args ...interface{})    {}
func (m *mockLogger) Printf(format string, args ...interface{})   {}
func (m *mockLogger) Warnf(format string, args ...interface{})    {}
func (m *mockLogger) Warningf(format string, args ...interface{}) {}
func (m *mockLogger) Errorf(format string, args ...interface{})   {}
func (m *mockLogger) Fatalf(format string, args ...interface{})   {}
func (m *mockLogger) Panicf(format string, args ...interface{})   {}

func (m *mockLogger) Debug(args ...interface{})   {}
func (m *mockLogger) Info(args ...interface{})    {}
func (m *mockLogger) Print(args ...interface{})   {}
func (m *mockLogger) Warn(args ...interface{})    {}
func (m *mockLogger) Warning(args ...interface{}) {}
func (m *mockLogger) Error(args ...interface{})   {}
func (m *mockLogger) Fatal(args ...interface{})   {}
func (m *mockLogger) Panic(args ...interface{})   {}

func (m *mockLogger) Debugln(args ...interface{})   {}
func (m *mockLogger) Infoln(args ...interface{})    {}
func (m *mockLogger) Println(args ...interface{})   {}
func (m *mockLogger) Warnln(args ...interface{})    {}
func (m *mockLogger) Warningln(args ...interface{}) {}
func (m *mockLogger) Errorln(args ...interface{})   {}
func (m *mockLogger) Fatalln(args ...interface{})   {}
func (m *mockLogger) Panicln(args ...interface{})   {}

type mockBackupHandler struct {
	tarExportingCalled bool
	tarExportedCalled  bool
	tarExportingResult error
	tarExportedResult  error
}

func (m *mockBackupHandler) OnTarExporting(path string) error {
	m.tarExportingCalled = true
	return m.tarExportingResult
}

func (m *mockBackupHandler) OnTarExported(path string, size int64) error {
	m.tarExportedCalled = true
	return m.tarExportedResult
}

func (m *mockBackupHandler) OnTagsFound(tags []string) error {
	return nil
}

func (m *mockBackupHandler) OnArtifactPulled(tag string, referrerCount int) error {
	return nil
}

func (m *mockBackupHandler) OnBackupCompleted(tagsCount int, path string, duration time.Duration) error {
	return nil
}

func (m *mockBackupHandler) Render() error {
	return nil
}
