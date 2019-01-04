package content

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/suite"
)

type ContentTestSuite struct {
	suite.Suite
}

var (
	testDirRoot, _ = filepath.Abs("../../.test")
)

func (suite *ContentTestSuite) TestStores() {
	memoryStore := NewMemoryStore()
	fileStore := NewFileStore(testDirRoot)

	ingesters := map[string]content.Ingester{
		"memory": memoryStore,
		"file":   fileStore,
	}

	providers := map[string]content.Provider{
		"memory": memoryStore,
		"file":   fileStore,
	}

	// Writers (Ingesters)
	for _, ingester := range ingesters {
		ctx := context.Background()
		refOpt := content.WithRef("localhost:5000/test1:latest")
		ingester.Writer(ctx, refOpt)

		// TODO: test writer.Write()
		/*
			writer, err := ingester.Writer(ctx, refOpt)
			suite.Nil(err, fmt.Sprintf("no error creating %s writer", key))
			suite.NotNil(writer)
		*/
	}

	// Readers (Providers)
	for _, provider := range providers {
		ctx := context.Background()
		configBytes := []byte("hello world")
		descriptor := ocispec.Descriptor{
			MediaType: ocispec.MediaTypeImageConfig,
			Digest:    digest.FromBytes(configBytes),
			Size:      int64(len(configBytes)),
		}
		provider.ReaderAt(ctx, descriptor)

		// TODO: test reader.ReadAt()
		/*
			reader, err := provider.ReaderAt(ctx, descriptor)
			suite.Nil(err, fmt.Sprintf("no error creating %s reader", key))
			suite.NotNil(reader)
		*/
	}
}

func TestContentTestSuite(t *testing.T) {
	suite.Run(t, new(ContentTestSuite))
}
