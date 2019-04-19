package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	_ "github.com/docker/distribution/registry/auth/htpasswd"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

var (
	testConfig   = "test.config"
	testHtpasswd = "test.htpasswd"
	testUsername = "alice"
	testPassword = "wonderland"
)

type DockerClientTestSuite struct {
	suite.Suite
	DockerRegistryHost string
	Client             *Client
	TempTestDir        string
}

func newContext() context.Context {
	return context.Background()
}

func (suite *DockerClientTestSuite) SetupSuite() {
	tempDir, err := ioutil.TempDir("", "oras_auth_docker_test")
	suite.Nil(err, "no error creating temp directory for test")
	suite.TempTestDir = tempDir

	// Create client
	client, err := NewClient(filepath.Join(suite.TempTestDir, testConfig))
	suite.Nil(err, "no error creating client")
	var ok bool
	suite.Client, ok = client.(*Client)
	suite.True(ok, "NewClient returns a *docker.Client inside")

	// Create htpasswd file with bcrypt
	secret, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	suite.Nil(err, "no error generating bcrypt password for test htpasswd file")
	authRecord := fmt.Sprintf("%s:%s\n", testUsername, string(secret))
	htpasswdPath := filepath.Join(suite.TempTestDir, testHtpasswd)
	err = ioutil.WriteFile(htpasswdPath, []byte(authRecord), 0644)
	suite.Nil(err, "no error creating test htpasswd file")

	// Registry config
	config := &configuration.Configuration{}
	port, err := freeport.GetFreePort()
	suite.Nil(err, "no error finding free port for test registry")
	suite.DockerRegistryHost = fmt.Sprintf("localhost:%d", port)
	config.HTTP.Addr = fmt.Sprintf(":%d", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	config.Auth = configuration.Auth{
		"htpasswd": configuration.Parameters{
			"realm": "localhost",
			"path":  htpasswdPath,
		},
	}
	dockerRegistry, err := registry.NewRegistry(context.Background(), config)
	suite.Nil(err, "no error finding free port for test registry")

	// Start Docker registry
	go dockerRegistry.ListenAndServe()
}

func (suite *DockerClientTestSuite) TearDownSuite() {
	os.RemoveAll(suite.TempTestDir)
}

func (suite *DockerClientTestSuite) Test_0_Login() {
	var err error

	err = suite.Client.Login(newContext(), suite.DockerRegistryHost, "oscar", "opponent")
	suite.NotNil(err, "error logging into registry with invalid credentials")

	err = suite.Client.Login(newContext(), suite.DockerRegistryHost, testUsername, testPassword)
	suite.Nil(err, "no error logging into registry with valid credentials")
}
func (suite *DockerClientTestSuite) Test_2_Logout() {
	var err error

	err = suite.Client.Logout(newContext(), "non-existing-host:42")
	suite.NotNil(err, "error logging out of registry that has no entry")

	err = suite.Client.Logout(newContext(), suite.DockerRegistryHost)
	suite.Nil(err, "no error logging out of registry")
}

func TestDockerClientTestSuite(t *testing.T) {
	suite.Run(t, new(DockerClientTestSuite))
}
