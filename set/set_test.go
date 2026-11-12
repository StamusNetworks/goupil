package set

import (
	"hash/maphash"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ Set[int] = Items[int]{}
	_ Set[int] = (*Safe[int])(nil)
	_ Set[int] = (*Evicting[int])(nil)
	_ Set[int] = (*Sharded[int])(nil)
)

type key struct {
	id   int
	name string
}

type impl[T comparable] struct {
	name       string
	new        func(t *testing.T) Set[T]
	concurrent bool
}

func hasher[T comparable]() func(T) uint64 {
	seed := maphash.MakeSeed()
	return func(v T) uint64 { return maphash.Comparable(seed, v) }
}

func impls[T comparable]() []impl[T] {
	return []impl[T]{
		{name: "items", new: func(*testing.T) Set[T] { return NewItems[T]() }},
		{name: "safe", new: func(*testing.T) Set[T] { return NewSafe[T]() }, concurrent: true},
		{name: "evicting", concurrent: true, new: func(t *testing.T) Set[T] {
			s, err := NewEvicting[T](128)
			require.NoError(t, err)
			return s
		}},
		{name: "sharded", concurrent: true, new: func(t *testing.T) Set[T] {
			s, err := NewSharded(4, 0, hasher[T]())
			require.NoError(t, err)
			return s
		}},
		{name: "sharded-evicting", concurrent: true, new: func(t *testing.T) Set[T] {
			s, err := NewSharded(4, 128, hasher[T]())
			require.NoError(t, err)
			return s
		}},
	}
}

func TestSetContract(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		testContract(t, []int{0, 1, 2, -3, 1 << 40})
	})
	t.Run("string", func(t *testing.T) {
		testContract(t, []string{"", "a", "bb", "ccc", "ünïcode"})
	})
	t.Run("key", func(t *testing.T) {
		testContract(t, []key{{}, {1, ""}, {0, "a"}, {1, "a"}, {2, "b"}})
	})
}

func testContract[T comparable](t *testing.T, vals []T) {
	require.GreaterOrEqual(t, len(vals), 5)

	for _, im := range impls[T]() {
		t.Run(im.name, func(t *testing.T) {
			t.Run("empty", func(t *testing.T) {
				s := im.new(t)
				assert.Equal(t, 0, s.Len())
				assert.Empty(t, s.Slice())
				assert.False(t, s.Has(vals[0]))
				assert.False(t, s.Delete(vals[0]))
			})

			t.Run("add", func(t *testing.T) {
				s := im.new(t)
				s.Add(vals[0])
				s.Add(vals[1])
				s.Add(vals[0])
				assert.Equal(t, 2, s.Len())
				assert.True(t, s.Has(vals[0]))
				assert.True(t, s.Has(vals[1]))
				assert.False(t, s.Has(vals[2]))
			})

			t.Run("delete", func(t *testing.T) {
				s := im.new(t)
				s.Add(vals[0])
				s.Add(vals[1])
				assert.True(t, s.Delete(vals[0]))
				assert.False(t, s.Delete(vals[0]))
				assert.False(t, s.Has(vals[0]))
				assert.True(t, s.Has(vals[1]))
				assert.Equal(t, 1, s.Len())
			})

			t.Run("load", func(t *testing.T) {
				s := im.new(t)
				assert.Equal(t, len(vals), s.Load(vals...))
				for _, v := range vals {
					assert.True(t, s.Has(v))
				}
				assert.Equal(t, len(vals), s.Len())
				assert.ElementsMatch(t, vals, s.Slice())
			})

			if !im.concurrent {
				return
			}
			t.Run("concurrent", func(t *testing.T) {
				s := im.new(t)
				var wg sync.WaitGroup
				for range 8 {
					wg.Go(func() {
						for i := range 1000 {
							v := vals[i%len(vals)]
							s.Add(v)
							s.Has(v)
							s.Len()
							if i%3 == 0 {
								s.Slice()
								s.Delete(v)
							}
						}
					})
				}
				wg.Wait()
				assert.Equal(t, len(s.Slice()), s.Len())
			})
		})
	}
}
