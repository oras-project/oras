package content

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/content/local"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// OCIStore provides content from the file system with the OCI-Image layout.
// Reference: https://github.com/opencontainers/image-spec/blob/master/image-layout.md
type OCIStore struct {
	content.Store

	root  string
	index *ocispec.Index
}

// NewOCIStore creates a new OCI store
func NewOCIStore(rootPath string) (*OCIStore, error) {
	fileStore, err := local.NewStore(rootPath)
	if err != nil {
		return nil, err
	}

	store := &OCIStore{
		Store: fileStore,
		root:  rootPath,
	}
	if err := store.validateOCILayoutFile(); err != nil {
		return nil, err
	}
	if err := store.LoadIndex(); err != nil {
		return nil, err
	}

	return store, nil
}

// LoadIndex reads the index.json from the file system
func (s *OCIStore) LoadIndex() error {
	path := filepath.Join(s.root, OCIImageIndexFile)
	indexFile, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		s.index = &ocispec.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2, // historical value
			},
		}

		return nil
	}
	defer indexFile.Close()

	return json.NewDecoder(indexFile).Decode(&s.index)
}

// SaveIndex writes the index.json to the file system
func (s *OCIStore) SaveIndex() error {
	indexJSON, err := json.Marshal(s.index)
	if err != nil {
		return err
	}

	path := filepath.Join(s.root, OCIImageIndexFile)
	return ioutil.WriteFile(path, indexJSON, 0644)
}

// validateOCILayoutFile ensures the `oci-layout` file
func (s *OCIStore) validateOCILayoutFile() error {
	layoutFilePath := filepath.Join(s.root, ocispec.ImageLayoutFile)
	layoutFile, err := os.Open(layoutFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		layout := ocispec.ImageLayout{
			Version: ocispec.ImageLayoutVersion,
		}
		layoutJSON, err := json.Marshal(layout)
		if err != nil {
			return err
		}

		return ioutil.WriteFile(layoutFilePath, layoutJSON, 0644)
	}
	defer layoutFile.Close()

	var layout *ocispec.ImageLayout
	err = json.NewDecoder(layoutFile).Decode(&layout)
	if err != nil {
		return err
	}
	if layout.Version != ocispec.ImageLayoutVersion {
		return ErrUnsupportedVersion
	}

	return nil
}
