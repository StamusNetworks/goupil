package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItems(t *testing.T) {
	s := NewItems[string]()
	s.Add("aaa")
	s.Add("bbb")
	s.Add("ccc")

	assert.Equal(t, 3, len(s))
	assert.True(t, s.Has("aaa"))
	assert.False(t, s.Has("zzz"))
	assert.Equal(t, 3, len(s.SliceNil()))
	assert.Equal(t, 2, len(s.Top(2)))
}
