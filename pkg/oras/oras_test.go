package oras

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	orascontent "github.com/deislabs/oras/pkg/content"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
)

var (
	testTarball  = "../../testdata/charts/chartmuseum-1.8.2.tgz"
	testDir      = "../../testdata/charts/chartmuseum"
	testDirFiles = []string{
		"Chart.yaml",
		"values.yaml",
		"README.md",
		"templates/_helpers.tpl",
		"templates/NOTES.txt",
		"templates/service.yaml",
		".helmignore",
	}
)

type ORASTestSuite struct {
	suite.Suite
	DockerRegistryHost string
}

func newContext() context.Context {
	return context.Background()
}

func newResolver() remotes.Resolver {
	return docker.NewResolver(docker.ResolverOptions{})
}

// Start Docker registry
func (suite *ORASTestSuite) SetupSuite() {
	config := &configuration.Configuration{}
	port, err := freeport.GetFreePort()
	if err != nil {
		suite.Nil(err, "no error finding free port for test registry")
	}
	suite.DockerRegistryHost = fmt.Sprintf("localhost:%d", port)
	config.HTTP.Addr = fmt.Sprintf(":%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	dockerRegistry, err := registry.NewRegistry(context.Background(), config)
	suite.Nil(err, "no error finding free port for test registry")

	go dockerRegistry.ListenAndServe()
}

// Push files to docker registry
func (suite *ORASTestSuite) Test_0_Push() {
	var (
		err         error
		ref         string
		desc        ocispec.Descriptor
		descriptors []ocispec.Descriptor
		store       *orascontent.FileStore
	)

	_, err = Push(newContext(), nil, ref, nil, descriptors)
	suite.NotNil(err, "error pushing with empty resolver")

	_, err = Push(newContext(), newResolver(), ref, nil, descriptors)
	suite.NotNil(err, "error pushing when context missing hostname")

	ref = fmt.Sprintf("%s/empty:test", suite.DockerRegistryHost)
	_, err = Push(newContext(), newResolver(), ref, nil, descriptors)
	suite.Nil(err, "no error pushing with empty descriptors")

	// Load descriptors with test chart tgz (as single layer)
	store = orascontent.NewFileStore("")
	basename := filepath.Base(testTarball)
	desc, err = store.Add(basename, "", testTarball)
	suite.Nil(err, "no error loading test chart")
	descriptors = []ocispec.Descriptor{desc}

	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	_, err = Push(newContext(), newResolver(), ref, store, descriptors)
	suite.Nil(err, "no error pushing test chart tgz (as single layer)")

	// Load descriptors with test chart dir (each file as layer)
	testDirAbs, err := filepath.Abs(testDir)
	suite.Nil(err, "no error parsing test directory")
	store = orascontent.NewFileStore(testDirAbs)
	descriptors = []ocispec.Descriptor{}
	var ff = func(pathX string, infoX os.FileInfo, errX error) error {
		if !infoX.IsDir() {
			filename := filepath.Join(filepath.Dir(pathX), infoX.Name())
			name := filepath.ToSlash(filename)
			desc, err = store.Add(name, "", filename)
			if err != nil {
				return err
			}
			descriptors = append(descriptors, desc)
		}
		return nil
	}

	cwd, _ := os.Getwd()
	os.Chdir(testDir)
	filepath.Walk(".", ff)
	os.Chdir(cwd)

	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	_, err = Push(newContext(), newResolver(), ref, store, descriptors)
	suite.Nil(err, "no error pushing test chart dir (each file as layer)")
}

// Pull files and verify descriptors
func (suite *ORASTestSuite) Test_1_Pull() {
	var (
		err         error
		ref         string
		descriptors []ocispec.Descriptor
		store       *orascontent.Memorystore
	)

	_, descriptors, err = Pull(newContext(), nil, ref, nil)
	suite.NotNil(err, "error pulling with empty resolver")
	suite.Nil(descriptors, "descriptors nil pulling with empty resolver")

	// Pull non-existant
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/nonexistant:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store)
	suite.NotNil(err, "error pulling non-existant ref")
	suite.Nil(descriptors, "descriptors empty with error")

	// Pull chart-tgz
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store)
	suite.Nil(err, "no error pulling chart-tgz ref")

	// Verify the descriptors, single layer/file
	content, err := ioutil.ReadFile(testTarball)
	suite.Nil(err, "no error loading test chart")
	name := filepath.Base(testTarball)
	_, actualContent, ok := store.GetByName(name)
	suite.True(ok, "find in memory")
	suite.Equal(content, actualContent, ".tgz content matches on pull")

	// Pull chart-dir
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store)
	suite.Nil(err, "no error pulling chart-dir ref")

	// Verify the descriptors, multiple layers/files
	cwd, _ := os.Getwd()
	os.Chdir(testDir)
	for _, filename := range testDirFiles {
		content, err = ioutil.ReadFile(filename)
		suite.Nil(err, fmt.Sprintf("no error loading %s", filename))
		_, actualContent, ok := store.GetByName(filename)
		suite.True(ok, "find in memory")
		suite.Equal(content, actualContent, fmt.Sprintf("%s content matches on pull", filename))
	}
	os.Chdir(cwd)
}

// Push and pull with customized media types
func (suite *ORASTestSuite) Test_2_MediaType() {
	var (
		testData = [][]string{
			{"hi.txt", "application/vnd.me.hi", "hi"},
			{"bye.txt", "application/vnd.me.bye", "bye"},
		}
		err         error
		ref         string
		descriptors []ocispec.Descriptor
		store       *orascontent.Memorystore
	)

	// Push content with customized media types
	store = orascontent.NewMemoryStore()
	descriptors = nil
	for _, data := range testData {
		desc := store.Add(data[0], data[1], []byte(data[2]))
		descriptors = append(descriptors, desc)
	}
	ref = fmt.Sprintf("%s/media-type:test", suite.DockerRegistryHost)
	_, err = Push(newContext(), newResolver(), ref, store, descriptors)
	suite.Nil(err, "no error pushing test data with customized media type")

	// Pull with all media types
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/media-type:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store)
	suite.Nil(err, "no error pulling media-type ref")
	suite.Equal(2, len(descriptors), "number of contents matches on pull")
	for _, data := range testData {
		_, actualContent, ok := store.GetByName(data[0])
		suite.True(ok, "find in memory")
		content := []byte(data[2])
		suite.Equal(content, actualContent, "test content matches on pull")
	}

	// Pull with specified media type
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/media-type:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store, WithAllowedMediaType(testData[0][1]))
	suite.Nil(err, "no error pulling media-type ref")
	suite.Equal(1, len(descriptors), "number of contents matches on pull")
	for _, data := range testData[:1] {
		_, actualContent, ok := store.GetByName(data[0])
		suite.True(ok, "find in memory")
		content := []byte(data[2])
		suite.Equal(content, actualContent, "test content matches on pull")
	}

	// Pull with non-existing media type
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/media-type:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store, WithAllowedMediaType("non.existing.media.type"))
	suite.Nil(err, "no error pulling media-type ref")
	suite.Equal(0, len(descriptors), "number of contents matches on pull")
}

// Pull with condition
func (suite *ORASTestSuite) Test_3_Conditional_Pull() {
	var (
		testData = [][]string{
			{"version.txt", "edge"},
			{"content.txt", "hello world"},
		}
		err         error
		ref         string
		descriptors []ocispec.Descriptor
		store       *orascontent.Memorystore
		stop        bool
	)

	// Push test content
	store = orascontent.NewMemoryStore()
	descriptors = nil
	for _, data := range testData {
		desc := store.Add(data[0], "", []byte(data[1]))
		descriptors = append(descriptors, desc)
	}
	ref = fmt.Sprintf("%s/conditional-pull:test", suite.DockerRegistryHost)
	_, err = Push(newContext(), newResolver(), ref, store, descriptors)
	suite.Nil(err, "no error pushing test data")

	// Pull all contents in sequence
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/conditional-pull:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store, WithPullByBFS)
	suite.Nil(err, "no error pulling ref")
	suite.Equal(2, len(descriptors), "number of contents matches on pull")
	for i, data := range testData {
		_, actualContent, ok := store.GetByName(data[0])
		suite.True(ok, "find in memory")
		content := []byte(data[1])
		suite.Equal(content, actualContent, "test content matches on pull")
		name, _ := orascontent.ResolveName(descriptors[i])
		suite.Equal(data[0], name, "content sequence matches on pull")
	}

	// Selective pull contents: stop at the very beginning
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/conditional-pull:test", suite.DockerRegistryHost)
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store, WithPullByBFS,
		WithPullBaseHandler(images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			if name, ok := orascontent.ResolveName(desc); ok && name == testData[0][0] {
				return nil, ErrStopProcessing
			}
			return nil, nil
		})))
	suite.Nil(err, "no error pulling ref")
	suite.Equal(0, len(descriptors), "number of contents matches on pull")

	// Selective pull contents: stop in the middle
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/conditional-pull:test", suite.DockerRegistryHost)
	stop = false
	_, descriptors, err = Pull(newContext(), newResolver(), ref, store, WithPullByBFS,
		WithPullBaseHandler(images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			if stop {
				return nil, ErrStopProcessing
			}
			if name, ok := orascontent.ResolveName(desc); ok && name == testData[0][0] {
				stop = true
			}
			return nil, nil
		})))
	suite.Nil(err, "no error pulling ref")
	suite.Equal(1, len(descriptors), "number of contents matches on pull")
	for _, data := range testData[:1] {
		_, actualContent, ok := store.GetByName(data[0])
		suite.True(ok, "find in memory")
		content := []byte(data[1])
		suite.Equal(content, actualContent, "test content matches on pull")
	}
}

func TestORASTestSuite(t *testing.T) {
	suite.Run(t, new(ORASTestSuite))
}
