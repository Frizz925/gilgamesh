package server

import (
	"testing"

	"github.com/Frizz925/gilgamesh/testutils/nettest"
	"github.com/Frizz925/gilgamesh/worker"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServer(t *testing.T) {
	require := require.New(t)
	logger, err := zap.NewDevelopment()
	require.NoError(err)

	l, c := nettest.NewListener()
	defer l.Close()
	defer c.Close()

	s := New(Config{
		Logger: logger,
		WorkerConfig: worker.Config{
			Logger: logger,
		},
	})
	s.Close()
}
