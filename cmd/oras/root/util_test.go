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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// MockOCIRegistry provides a minimal in-memory OCI registry
type MockOCIRegistry struct {
	server   *httptest.Server
	blobs    map[string][]byte // digest -> blob data
	manifest map[string][]byte // repo:tag -> manifest
	uploads  map[string][]byte // upload uuid -> partial data
	mu       sync.RWMutex
}

// NewMockOCIRegistry creates a new mock registry
func NewMockOCIRegistry() *MockOCIRegistry {
	registry := &MockOCIRegistry{
		blobs:    make(map[string][]byte),
		manifest: make(map[string][]byte),
		uploads:  make(map[string][]byte),
	}

	mux := http.NewServeMux()

	// Health check
	//mux.HandleFunc("/v2/", registry.handleVersion)

	// Blob operations
	mux.HandleFunc("/v2/", registry.handleRequest)

	registry.server = httptest.NewServer(mux)
	return registry
}

// URL returns the registry URL
func (r *MockOCIRegistry) URL() string {
	return r.server.URL
}

// Close shuts down the registry
func (r *MockOCIRegistry) Close() {
	r.server.Close()
}

// handleVersion handles /v2/ endpoint
func (r *MockOCIRegistry) handleVersion(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/v2/" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		return
	}
	r.handleRequest(w, req)
}

// handleRequest routes requests to appropriate handlers
func (r *MockOCIRegistry) handleRequest(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	switch {
	case strings.Contains(path, "/uploads/"):
		r.handleUpload(w, req)
	case strings.Contains(path, "/blobs/"):
		r.handleBlob(w, req)
	case strings.Contains(path, "/manifests/"):
		r.handleManifest(w, req)
	default:
		http.NotFound(w, req)
	}
}

// handleBlob handles blob operations
func (r *MockOCIRegistry) handleBlob(w http.ResponseWriter, req *http.Request) {
	// Extract digest from path like /v2/repo/blobs/sha256:abc123
	parts := strings.Split(req.URL.Path, "/blobs/")
	if len(parts) != 2 {
		http.Error(w, "Invalid blob path", http.StatusBadRequest)
		return
	}

	digest := parts[1]

	switch req.Method {
	case "HEAD":
		r.mu.RLock()
		data, exists := r.blobs[digest]
		r.mu.RUnlock()

		if !exists {
			http.Error(w, "Blob not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

	case "GET":
		r.mu.RLock()
		data, exists := r.blobs[digest]
		r.mu.RUnlock()

		if !exists {
			http.Error(w, "Blob not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleManifest handles manifest operations
func (r *MockOCIRegistry) handleManifest(w http.ResponseWriter, req *http.Request) {
	// Extract repo and tag from path like /v2/repo/manifests/tag
	parts := strings.Split(req.URL.Path, "/manifests/")
	if len(parts) != 2 {
		http.Error(w, "Invalid manifest path", http.StatusBadRequest)
		return
	}

	repo := strings.TrimPrefix(parts[0], "/v2/")
	tag := parts[1]
	key := repo + ":" + tag

	switch req.Method {
	case "HEAD":
		r.mu.RLock()
		data, exists := r.manifest[key]
		r.mu.RUnlock()

		if !exists {
			http.Error(w, "Manifest not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.WriteHeader(http.StatusOK)

	case "GET":
		r.mu.RLock()
		data, exists := r.manifest[key]
		r.mu.RUnlock()

		if !exists {
			http.Error(w, "Manifest not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.Write(data)

	case "PUT":
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		r.mu.Lock()
		r.manifest[key] = body
		r.mu.Unlock()

		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUpload handles blob upload operations
func (r *MockOCIRegistry) handleUpload(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		// Start upload - extract repo from path like /v2/repo/blobs/uploads/
		path := req.URL.Path
		if !strings.Contains(path, "/blobs/uploads/") {
			http.Error(w, "Invalid upload path", http.StatusBadRequest)
			return
		}

		// Generate upload UUID
		uploadUUID := "upload-" + fmt.Sprintf("%d", len(r.uploads))

		r.mu.Lock()
		r.uploads[uploadUUID] = make([]byte, 0)
		r.mu.Unlock()

		// Return upload location
		repo := strings.Split(strings.TrimPrefix(path, "/v2/"), "/blobs/uploads/")[0]
		uploadURL := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uploadUUID)

		w.Header().Set("Location", uploadURL)
		w.Header().Set("Range", "0-0")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusAccepted)

	case "PATCH":
		// Upload chunk - extract UUID from path like /v2/repo/blobs/uploads/uuid
		parts := strings.Split(req.URL.Path, "/uploads/")
		if len(parts) != 2 {
			http.Error(w, "Invalid upload path", http.StatusBadRequest)
			return
		}

		uploadUUID := parts[1]

		// Read chunk data
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		r.mu.Lock()
		existing, exists := r.uploads[uploadUUID]
		if !exists {
			r.mu.Unlock()
			http.Error(w, "Upload not found", http.StatusNotFound)
			return
		}

		// Append to existing data
		r.uploads[uploadUUID] = append(existing, body...)
		newSize := len(r.uploads[uploadUUID])
		r.mu.Unlock()

		w.Header().Set("Location", req.URL.Path)
		w.Header().Set("Range", fmt.Sprintf("0-%d", newSize-1))
		w.WriteHeader(http.StatusAccepted)

	case "PUT":
		// Complete upload
		digest := req.URL.Query().Get("digest")
		if digest == "" {
			http.Error(w, "Missing digest", http.StatusBadRequest)
			return
		}

		// Extract UUID from path
		parts := strings.Split(req.URL.Path, "/uploads/")
		if len(parts) != 2 {
			http.Error(w, "Invalid upload path", http.StatusBadRequest)
			return
		}

		uploadUUID := parts[1]

		r.mu.Lock()
		uploadData, exists := r.uploads[uploadUUID]
		if !exists {
			r.mu.Unlock()
			http.Error(w, "Upload not found", http.StatusNotFound)
			return
		}

		// Read final chunk if any
		body, err := io.ReadAll(req.Body)
		if err != nil {
			r.mu.Unlock()
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Combine all data
		finalData := append(uploadData, body...)

		// Store blob and cleanup upload
		r.blobs[digest] = finalData
		delete(r.uploads, uploadUUID)
		r.mu.Unlock()

		w.Header().Set("Location", strings.Replace(req.URL.Path, "/uploads/"+uploadUUID, "/blobs/"+digest, 1))
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
