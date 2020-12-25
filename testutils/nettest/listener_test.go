package nettest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestListener(t *testing.T) {
	require := require.New(t)
	server, client := NewListener()
	expected := []byte("message to be sent")
	ch := make(chan []byte, 1)

	var g errgroup.Group
	g.Go(func() error {
		_, err := client.Write(expected)
		return err
	})
	g.Go(func() error {
		c, err := server.Accept()
		if err != nil {
			return err
		}
		b := make([]byte, 512)
		n, err := c.Read(b)
		if err != nil {
			return err
		}
		ch <- b[:n]
		return nil
	})

	require.NoError(g.Wait())
	require.Equal(expected, <-ch)
	require.NotNil(server.Addr())
	require.NoError(server.Close())
}
