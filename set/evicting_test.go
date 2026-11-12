package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvictingInvalidSize(t *testing.T) {
	_, err := NewEvicting[int](0)
	assert.Error(t, err)
}

func TestEvictingBounded(t *testing.T) {
	e, err := NewEvicting[int](4)
	require.NoError(t, err)
	for i := range 100 {
		e.Add(i)
	}
	assert.Equal(t, 4, e.Len())
}

func TestEvictingKeepsHeavyHitter(t *testing.T) {
	e, err := NewEvicting[int](8)
	require.NoError(t, err)

	e.Add(-1)
	e.Add(-1)
	for i := range 1000 {
		e.Add(i)
	}
	assert.True(t, e.Has(-1))
}
