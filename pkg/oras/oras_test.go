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
	Resolver           remotes.Resolver
	DockerRegistryHost string
}

func (suite *ORASTestSuite) SetupSuite() {
	suite.Resolver = docker.NewResolver(docker.ResolverOptions{})

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

func (suite *ORASTestSuite) TestPush() {
	var err error
	var ctx context.Context
	var resolver remotes.Resolver
	var ref string
	var contents map[string][]byte

	err = Push(ctx, nil, ref, contents)
	suite.NotNil(err, "error pushing with empty resolver")

	ctx = context.Background()
	resolver = docker.NewResolver(docker.ResolverOptions{})
	err = Push(ctx, resolver, ref, contents)
	suite.NotNil(err, "error pushing when context missing hostname")

	ref = fmt.Sprintf("%s/empty", suite.DockerRegistryHost)
	err = Push(ctx, resolver, ref, contents)
	suite.Nil(err, "no error pushing with empty contents")

	// Load contents with test chart tgz (as single layer)
	contents = make(map[string][]byte)
	content, err := ioutil.ReadFile(testChartTgz)
	suite.Nil(err, "no error loading test chart")
	basename := filepath.Base(testChartTgz)
	contents[basename] = content

	ref = fmt.Sprintf("%s/chart-tgz", suite.DockerRegistryHost)
	err = Push(ctx, resolver, ref, contents)
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

	ref = fmt.Sprintf("%s/chart-dir", suite.DockerRegistryHost)
	err = Push(ctx, resolver, ref, contents)
	suite.Nil(err, "no error pushing test chart dir (each file as layer)")
}

func (suite *ORASTestSuite) TestPull() {
	var err error
	var ctx context.Context
	var ref string
	var contents map[string][]byte

	contents, err = Pull(ctx, nil, ref)
	suite.NotNil(err, "error pulling with empty resolver")
	suite.Nil(contents, "contents nil pulling with empty resolver")
}

func TestPushTestSuite(t *testing.T) {
	suite.Run(t, new(ORASTestSuite))
}
