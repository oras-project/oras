package oras

func (suite *PushOptsSuite) TestValidateNameAsPathWindows() {
	var err error

	// disallow absolute path
	err = ValidateNameAsPath(descFromName("C:\\foo\\bar"))
	suite.Error(err, "windows: absolute path disallowed")
}
