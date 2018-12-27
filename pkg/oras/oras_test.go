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
	var contents map[string][]byte

	err = Push(newContext(), nil, ref, contents)
	suite.NotNil(err, "error pushing with empty resolver")

	err = Push(newContext(), newResolver(), ref, contents)
	suite.NotNil(err, "error pushing when context missing hostname")

	ref = fmt.Sprintf("%s/empty:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, contents)
	suite.NotNil(ErrEmptyContents, err, "error pushing with empty contents")

	// Load contents with test chart tgz (as single layer)
	contents = make(map[string][]byte)
	content, err := ioutil.ReadFile(testTarball)
	suite.Nil(err, "no error loading test chart")
	basename := filepath.Base(testTarball)
	contents[basename] = content

	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, contents)
	suite.Nil(err, "no error pushing test chart tgz (as single layer)")

	// Load contents with test chart dir (each file as layer)
	contents = make(map[string][]byte)
	var ff = func(pathX string, infoX os.FileInfo, errX error) error {
		if !infoX.IsDir() {
			filename := filepath.Join(filepath.Dir(pathX), infoX.Name())
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				return err
			}
			contents[filename] = content
		}
		return nil
	}

	cwd, _ := os.Getwd()
	os.Chdir(testDir)
	filepath.Walk(".", ff)
	os.Chdir(cwd)

	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	err = Push(newContext(), newResolver(), ref, contents)
	suite.Nil(err, "no error pushing test chart dir (each file as layer)")
}

// Pull refs
func (suite *ORASTestSuite) Test_1_Pull() {
	var err error
	var ref string
	var contents map[string][]byte

	contents, err = Pull(newContext(), nil, ref)
	suite.NotNil(err, "error pulling with empty resolver")
	suite.Nil(contents, "contents nil pulling with empty resolver")

	// Pull non-existant
	ref = fmt.Sprintf("%s/nonexistant:test", suite.DockerRegistryHost)
	contents, err = Pull(newContext(), newResolver(), ref)
	suite.NotNil(err, "error pulling non-existant ref")
	suite.Nil(contents, "contents empty with error")

	// Pull chart-tgz
	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	contents, err = Pull(newContext(), newResolver(), ref)
	suite.Nil(err, "no error pulling chart-tgz ref")

	// Verify the contents, single layer/file
	content, err := ioutil.ReadFile(testTarball)
	suite.Nil(err, "no error loading test chart")
	suite.Equal(content, contents[filepath.Base(testTarball)], ".tgz content matches on pull")

	// Pull chart-dir
	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	contents, err = Pull(newContext(), newResolver(), ref)
	suite.Nil(err, "no error pulling chart-dir ref")

	// Verify the contents, multiple layers/files
	cwd, _ := os.Getwd()
	os.Chdir(testDir)
	for _, filename := range testDirFiles {
		content, err = ioutil.ReadFile(filename)
		suite.Nil(err, fmt.Sprintf("no error loading %s", filename))
		suite.Equal(content, contents[filename], fmt.Sprintf("%s content matches on pull", filename))
	}
	os.Chdir(cwd)
}

func TestORASTestSuite(t *testing.T) {
	suite.Run(t, new(ORASTestSuite))
}
