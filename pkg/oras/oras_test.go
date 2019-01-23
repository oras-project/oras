package oras

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/phayes/freeport"
	orascontent "github.com/deislabs/oras/pkg/content"
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

	err = Push(newContext(), nil, ref, nil, descriptors)
	suite.NotNil(err, "error pushing with empty resolver")

	err = Push(newContext(), newResolver(), ref, nil, descriptors)
	suite.NotNil(err, "error pushing when context missing hostname")

	ref = fmt.Sprintf("%s/empty:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, nil, descriptors)
	suite.NotNil(ErrEmptyDescriptors, err, "error pushing with empty descriptors")

	// Load descriptors with test chart tgz (as single layer)
	store = orascontent.NewFileStore("")
	basename := filepath.Base(testTarball)
	desc, err = store.Add(basename, "", testTarball)
	suite.Nil(err, "no error loading test chart")
	descriptors = []ocispec.Descriptor{desc}

	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, store, descriptors)
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
	err = Push(newContext(), newResolver(), ref, store, descriptors)
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

	descriptors, err = Pull(newContext(), nil, ref, nil)
	suite.NotNil(err, "error pulling with empty resolver")
	suite.Nil(descriptors, "descriptors nil pulling with empty resolver")

	// Pull non-existant
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/nonexistant:test", suite.DockerRegistryHost)
	descriptors, err = Pull(newContext(), newResolver(), ref, store)
	suite.NotNil(err, "error pulling non-existant ref")
	suite.Nil(descriptors, "descriptors empty with error")

	// Pull chart-tgz
	store = orascontent.NewMemoryStore()
	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	descriptors, err = Pull(newContext(), newResolver(), ref, store)
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
	descriptors, err = Pull(newContext(), newResolver(), ref, store)
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

func TestORASTestSuite(t *testing.T) {
	suite.Run(t, new(ORASTestSuite))
}
