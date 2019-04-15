package oras

import (
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/suite"
)

type PushOptsSuite struct {
	suite.Suite
}

func (suite *PushOptsSuite) TestValidateNameAsPath() {
	var err error

	// valid path
	err = ValidateNameAsPath(descFromName("hello.txt"))
	suite.NoError(err, "valid path")
	err = ValidateNameAsPath(descFromName("foo/bar"))
	suite.NoError(err, "valid path with multiple sub-directories")

	// no empty name
	err = ValidateNameAsPath(descFromName(""))
	suite.Error(err, "empty path")

	// path should be clean
	err = ValidateNameAsPath(descFromName("./hello.txt"))
	suite.Error(err, "dirty path")
	err = ValidateNameAsPath(descFromName("foo/../bar"))
	suite.Error(err, "dirty path")

	// path should be slash-separated
	err = ValidateNameAsPath(descFromName("foo\\bar"))
	suite.Error(err, "path not slash separated")

	// disallow absolute path
	err = ValidateNameAsPath(descFromName("/foo/bar"))
	suite.Error(err, "unix: absolute path disallowed")
	err = ValidateNameAsPath(descFromName("C:\\foo\\bar"))
	suite.Error(err, "windows: absolute path disallowed")
	err = ValidateNameAsPath(descFromName("C:/foo/bar"))
	suite.Error(err, "windows: absolute path disallowed")

	// disallow path traversal
	err = ValidateNameAsPath(descFromName(".."))
	suite.Error(err, "path traversal disallowed")
	err = ValidateNameAsPath(descFromName("../bar"))
	suite.Error(err, "path traversal disallowed")
	err = ValidateNameAsPath(descFromName("foo/../../bar"))
	suite.Error(err, "path traversal disallowed")
}

func TestPushOptsSuite(t *testing.T) {
	suite.Run(t, new(PushOptsSuite))
}

func descFromName(name string) ocispec.Descriptor {
	return ocispec.Descriptor{
		Annotations: map[string]string{
			ocispec.AnnotationTitle: name,
		},
	}
}
