package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemsExtend(t *testing.T) {
	s := NewItems[int]()
	s.Load(1, 2)
	s.Extend(Items[int]{2: true, 3: true})
	assert.ElementsMatch(t, []int{1, 2, 3}, s.Slice())
}

func TestItemsSliceNil(t *testing.T) {
	assert.Nil(t, NewItems[int]().SliceNil())

	s := NewItems[int]()
	s.Add(1)
	assert.Equal(t, []int{1}, s.SliceNil())
}

func TestItemsTop(t *testing.T) {
	s := NewItems[int]()
	s.Load(1, 2, 3)

	assert.Len(t, s.Top(2), 2)
	assert.Subset(t, s.Slice(), s.Top(2))
	assert.Len(t, s.Top(-1), 3)
	assert.Len(t, s.Top(10), 3)
	assert.Empty(t, NewItems[int]().Top(1))
}

func TestItemsCopy(t *testing.T) {
	s := Items[int]{1: true, 2: false}
	c := s.Copy()
	assert.Equal(t, Items[int]{1: true}, c)

	c.Add(3)
	assert.False(t, s.Has(3))
}

func TestItemsDropRandom(t *testing.T) {
	s := NewItems[int]()
	s.Load(1, 2, 3)
	s.DropRandom()
	assert.Equal(t, 2, s.Len())
}
