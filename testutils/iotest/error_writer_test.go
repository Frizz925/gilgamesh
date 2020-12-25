package iotest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorWriter(t *testing.T) {
	require := require.New(t)
	expected := errors.New("expected error")
	ew := NewErrorWriter(expected)
	n, err := ew.Write(nil)
	require.Equal(0, n)
	require.Equal(expected, err)
}
