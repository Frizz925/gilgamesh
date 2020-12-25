package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFull(t *testing.T) {
	require := require.New(t)
	payload := []byte("payloadtowrite")
	buf := &bytes.Buffer{}
	require.NoError(WriteFull(buf, payload))
	require.Equal(payload, buf.Bytes())
}
