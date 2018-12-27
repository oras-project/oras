package oras

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/phayes/freeport"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	"github.com/stretchr/testify/suite"
)

var (
	testChartDir = "../../testdata/charts/chartmuseum"
	testChartTgz = "../../testdata/charts/chartmuseum-1.8.2.tgz"
)

type ORASTestSuite struct {
	suite.Suite
	Context context.Context
	DockerRegistryHost string
}

func newResolver() remotes.Resolver{
	return docker.NewResolver(docker.ResolverOptions{})
}

func (suite *ORASTestSuite) SetupSuite() {
	suite.Context = context.Background()

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

	// Start Docker registry
	go dockerRegistry.ListenAndServe()
}

func (suite *ORASTestSuite) TearDownSuite() {
	return
}

func (suite *ORASTestSuite) Test_0_Push() {
	var err error
	var ref string
	var contents map[string][]byte

	err = Push(suite.Context, nil, ref, contents)
	suite.NotNil(err, "error pushing with empty resolver")

	err = Push(suite.Context, newResolver(), ref, contents)
	suite.NotNil(err, "error pushing when context missing hostname")

	ref = fmt.Sprintf("%s/empty:test", suite.DockerRegistryHost)
	err = Push(suite.Context, newResolver(), ref, contents)
	suite.NotNil(err, "error pushing with empty contents")

	// Load contents with test chart tgz (as single layer)
	contents = make(map[string][]byte)
	content, err := ioutil.ReadFile(testChartTgz)
	suite.Nil(err, "no error loading test chart")
	basename := filepath.Base(testChartTgz)
	contents[basename] = content

	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	err = Push(suite.Context, newResolver(), ref, contents)
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

	os.Chdir(testChartDir)
	filepath.Walk(".", ff)

	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	err = Push(suite.Context, newResolver(), ref, contents)
	suite.Nil(err, "no error pushing test chart dir (each file as layer)")
}

func (suite *ORASTestSuite) Test_1_Pull() {
	var err error
	var ref string
	var contents map[string][]byte

	contents, err = Pull(suite.Context, nil, ref)
	suite.NotNil(err, "error pulling with empty resolver")
	suite.Nil(contents, "contents nil pulling with empty resolver")

	// Pull non-existant
	ref = fmt.Sprintf("%s/nonexistant:test", suite.DockerRegistryHost)
	contents, err = Pull(suite.Context, newResolver(), ref)
	suite.NotNil(err, "error pulling non-existant ref")
	suite.Nil(contents, "contents empty with error")

	// Pull chart-tgz
	ref = fmt.Sprintf("%s/chart-tgz:test", suite.DockerRegistryHost)
	_, err = Pull(suite.Context, newResolver(), ref)
	suite.Nil(err, "no error pulling chart-tgz ref")

	// Pull chart-dir
	ref = fmt.Sprintf("%s/chart-dir:test", suite.DockerRegistryHost)
	_, err = Pull(suite.Context, newResolver(), ref)
	suite.Nil(err, "no error pulling chart-dir ref")
}

func TestORASTestSuite(t *testing.T) {
	suite.Run(t, new(ORASTestSuite))
}
