package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomicBool(t *testing.T) {
	assert := assert.New(t)
	var ab AtomicBool
	assert.False(ab.Get())
	ab.Set(true)
	assert.True(ab.Get())
}
