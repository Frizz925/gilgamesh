package worker

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type PoolTestSuite struct {
	suite.Suite
	config Config
}

func TestPool(t *testing.T) {
	suite.Run(t, &PoolTestSuite{})
}

func (suite *PoolTestSuite) SetupSuite() {
	logger, err := zap.NewDevelopment()
	suite.Require().NoError(err)
	suite.config.Logger = logger
}

func (suite *PoolTestSuite) TestPreallocated() {
	p := NewPool(16, suite.config)
	w := p.Get()
	p.Put(w)
	p.Close()
}

func (suite *PoolTestSuite) TestDynamic() {
	p := NewPool(0, suite.config)
	w := p.Get()
	p.Put(w)
	p.Close()
}
