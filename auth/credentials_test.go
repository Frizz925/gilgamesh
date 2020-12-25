package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentials(t *testing.T) {
	require := require.New(t)
	username := "user"
	password := []byte("deadbeef")

	pw, err := CreatePassword(password)
	require.NoError(err)
	require.NoError(pw.Compare(password))

	buf := &bytes.Buffer{}
	require.NoError(WriteCredentials(buf, Credentials{username: pw}))
	creds, err := ReadCredentials(buf)
	require.NoError(err)

	pw, ok := creds[username]
	require.True(ok)
	require.NoError(pw.Compare(password))
}
