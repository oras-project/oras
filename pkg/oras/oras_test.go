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
	var err error
	var ref string
	var blobs map[string]Blob

	err = Push(newContext(), nil, ref, blobs)
	suite.NotNil(err, "error pushing with empty resolver")

	err = Push(newContext(), newResolver(), ref, blobs)
	suite.NotNil(err, "error pushing when context missing hostname")

	ref = fmt.Sprintf("%s/empty:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, blobs)
	suite.NotNil(ErrEmptyBlobs, err, "error pushing with empty blobs")

	// Load blobs with test chart tgz (as single layer)
	blobs = make(map[string]Blob)
	content, err := ioutil.ReadFile(testTarball)
	suite.Nil(err, "no error loading test chart")
	basename := filepath.Base(testTarball)
	blobs[basename] = Blob{Content: content}

	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, blobs)
	suite.Nil(err, "no error pushing test chart tgz (as single layer)")

	// Load blobs with test chart dir (each file as layer)
	blobs = make(map[string]Blob)
	var ff = func(pathX string, infoX os.FileInfo, errX error) error {
		if !infoX.IsDir() {
			filename := filepath.Join(filepath.Dir(pathX), infoX.Name())
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				return err
			}
			blobs[filepath.ToSlash(filename)] = Blob{Content: content}
		}
		return nil
	}

	cwd, _ := os.Getwd()
	os.Chdir(testDir)
	filepath.Walk(".", ff)
	os.Chdir(cwd)

	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, blobs)
	suite.Nil(err, "no error pushing test chart dir (each file as layer)")
}

// Pull files and verify blobs
func (suite *ORASTestSuite) Test_1_Pull() {
	var err error
	var ref string
	var blobs map[string]Blob

	blobs, err = Pull(newContext(), nil, ref)
	suite.NotNil(err, "error pulling with empty resolver")
	suite.Nil(blobs, "blobs nil pulling with empty resolver")

	// Pull non-existant
	ref = fmt.Sprintf("%s/nonexistant:test", suite.DockerRegistryHost)
	blobs, err = Pull(newContext(), newResolver(), ref)
	suite.NotNil(err, "error pulling non-existant ref")
	suite.Nil(blobs, "blobs empty with error")

	// Pull chart-tgz
	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	blobs, err = Pull(newContext(), newResolver(), ref)
	suite.Nil(err, "no error pulling chart-tgz ref")

	// Verify the blobs, single layer/file
	content, err := ioutil.ReadFile(testTarball)
	suite.Nil(err, "no error loading test chart")
	suite.Equal(content, blobs[filepath.Base(testTarball)].Content, ".tgz content matches on pull")

	// Pull chart-dir
	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	blobs, err = Pull(newContext(), newResolver(), ref)
	suite.Nil(err, "no error pulling chart-dir ref")

	// Verify the blobs, multiple layers/files
	cwd, _ := os.Getwd()
	os.Chdir(testDir)
	for _, filename := range testDirFiles {
		content, err = ioutil.ReadFile(filename)
		suite.Nil(err, fmt.Sprintf("no error loading %s", filename))
		suite.Equal(content, blobs[filename].Content, fmt.Sprintf("%s content matches on pull", filename))
	}
	os.Chdir(cwd)
}

func TestORASTestSuite(t *testing.T) {
	suite.Run(t, new(ORASTestSuite))
}
