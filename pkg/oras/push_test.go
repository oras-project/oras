package oras

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/phayes/freeport"
	"testing"
	"time"

	"github.com/containerd/containerd/remotes"
	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	"github.com/stretchr/testify/suite"
)

type PushTestSuite struct {
	suite.Suite
	DockerRegistryHost string
}

func (suite *PushTestSuite) SetupSuite() {
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

func (suite *PushTestSuite) TearDownSuite() {
	return
}

func (suite *PushTestSuite) TestPush() {
	var err error
	var ctx context.Context
	var resolver remotes.Resolver
	var ref string
	var contents map[string][]byte

	err = Push(ctx, resolver, ref, contents)
	suite.NotNil(err, "error with empty resolver")

	ctx = context.Background()
	resolver = docker.NewResolver(docker.ResolverOptions{})
	err = Push(ctx, resolver, ref, contents)
	suite.NotNil(err, "error when context missing hostname")

	ref = fmt.Sprintf("%s/myapp", suite.DockerRegistryHost)
	err = Push(ctx, resolver, ref, contents)
	suite.NotNil(err, "error with empty contents")
}

func TestPushTestSuite(t *testing.T) {
	suite.Run(t, new(PushTestSuite))
}
