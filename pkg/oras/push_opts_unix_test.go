// +build !windows

package oras

func (suite *PushOptsSuite) TestValidateNameAsPathUnix() {
	var err error

	// disallow absolute path
	err = ValidateNameAsPath(descFromName("/foo/bar"))
	suite.Error(err, "unix: absolute path disallowed")
}
