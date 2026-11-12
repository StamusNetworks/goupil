package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func identity(i int) uint64 { return uint64(i) } //nolint:gosec

func TestShardedRouting(t *testing.T) {
	s, err := NewSharded(3, 0, identity)
	require.NoError(t, err)

	s.Load(0, 1, 2, 3, 4)
	assert.Equal(t, []int{2, 2, 1}, s.Distribution())

	s.Delete(3)
	assert.Equal(t, []int{1, 2, 1}, s.Distribution())
}

func TestShardedInvalidConfig(t *testing.T) {
	_, err := NewSharded[int](2, 0, nil)
	assert.Error(t, err)

	_, err = NewSharded(0, 0, identity)
	assert.Error(t, err)

	// size smaller than shard count leaves shards with zero capacity
	_, err = NewSharded(4, 2, identity)
	assert.Error(t, err)
}

func TestShardedEvictsPerShard(t *testing.T) {
	s, err := NewSharded(2, 10, hasher[int]())
	require.NoError(t, err)

	for i := range 1000 {
		s.Add(i)
	}
	assert.LessOrEqual(t, s.Len(), 10)
	for _, n := range s.Distribution() {
		assert.LessOrEqual(t, n, 5)
	}
}
