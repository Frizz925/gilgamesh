package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsProduction(t *testing.T) {
	require := require.New(t)
	require.NoError(os.Setenv("ENV", "test"))
	require.False(IsProduction())
}
