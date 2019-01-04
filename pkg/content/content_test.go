package content

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ContentTestSuite struct {
	suite.Suite
}

func (suite *ContentTestSuite) TestContent() {
	return
}

func TestContentTestSuite(t *testing.T) {
	suite.Run(t, new(ContentTestSuite))
}
