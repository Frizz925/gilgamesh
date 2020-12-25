package utils

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Frizz925/gilgamesh/testutils/iotest"
	"github.com/stretchr/testify/require"
)

func TestWriteFull(t *testing.T) {
	require := require.New(t)
	payload := []byte("payloadtowrite")
	buf := &bytes.Buffer{}
	require.NoError(WriteFull(buf, payload))
	require.Equal(payload, buf.Bytes())

	expectedErr := errors.New("expected error")
	ew := iotest.NewErrorWriter(expectedErr)
	require.Equal(expectedErr, WriteFull(ew, payload))
}
