package dict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var ()

func TestEntry_Get_ArrayTraversal(t *testing.T) {
	// Simulates EVE JSON: dns.query is an array of objects
	entry := Entry{
		"dns": Entry{
			"query": []any{
				map[string]any{"rrname": "germakhya.xyz", "rrtype": "A"},
				map[string]any{"rrname": "example.com", "rrtype": "AAAA"},
			},
		},
	}

	// Traversing through array should collect values from each element
	val, ok := entry.Get("dns", "query", "rrname")
	assert.True(t, ok)
	assert.Equal(t, []any{"germakhya.xyz", "example.com"}, val)

	// Single key inside array
	val, ok = entry.Get("dns", "query", "rrtype")
	assert.True(t, ok)
	assert.Equal(t, []any{"A", "AAAA"}, val)

	// No remaining keys returns the array as-is (existing behavior)
	val, ok = entry.Get("dns", "query")
	assert.True(t, ok)
	arr, isSlice := val.([]any)
	assert.True(t, isSlice)
	assert.Len(t, arr, 2)

	// Key not present in array elements returns false
	_, ok = entry.Get("dns", "query", "nonexistent")
	assert.False(t, ok)

	// Deeper nesting through arrays
	deep := Entry{
		"a": []any{
			map[string]any{"b": map[string]any{"c": "found"}},
		},
	}
	val, ok = deep.Get("a", "b", "c")
	assert.True(t, ok)
	assert.Equal(t, []any{"found"}, val)
}

func TestEntry_Keys(t *testing.T) {
	reference := Entry{
		"a": 1,
		"b": true,
		"c": "asdf",
		"d": Entry{
			"e": 11233.5,
			"f": Entry{
				"g": "ashdkljasa",
			},
		},
		"h": []string{"a", "", "cv"},
		"i": Entry{
			"j": []map[string]any{
				{"a": 1, "b": true, "c": "1234"},
				{"a": 3, "b": false, "c": "lalal"},
			},
		},
	}
	expectedKeys := []string{"a", "b", "c", "d", "h", "i"}
	expectedKeysRecurse := []string{"a", "b", "c", "d.e", "d.f.g", "h", "i.j"}

	assert.Equal(t, expectedKeys, reference.Keys(true))
	assert.Equal(t, expectedKeysRecurse, reference.KeysRecurse(true))
}
