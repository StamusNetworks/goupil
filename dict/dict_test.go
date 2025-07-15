package dict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var ()

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
