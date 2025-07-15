package dict

import (
	"maps"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Entry map[string]any

// GetWithDotKey is a recursive helper for doing dot notation lookups into nested maps
// Does not assert any concrete type. This method is also VERY slow compared to subsequent Get
// method as it splits key string on each recursion, resulting in five times slower operation.
// This method only exists for benchmarking and regular Get should be used instead.
func (d Entry) GetWithDotKey(key string) (any, bool) {
	bits := strings.SplitN(key, ".", 2)
	if len(bits) == 0 {
		return nil, false
	}
	if val, ok := d[bits[0]]; ok {
		switch res := val.(type) {
		case map[string]any:
			// Result is another dynamic map, recurse with key remainder
			return Entry(res).GetWithDotKey(bits[1])
		case Entry:
			return res.GetWithDotKey(bits[1])
		default:
			return res, ok
		}
	}
	return nil, false
}

// Get is a helper for doing recursive lookups into nested maps (nested JSON). Key argument is a slice of strings
func (d Entry) Get(key ...string) (any, bool) {
	if len(key) == 0 {
		return nil, false
	} else if len(key) == 1 {
		if val, ok := d[key[0]]; ok {
			return val, ok
		}
		return nil, false
	}
	if val, ok := d[key[0]]; ok {
		switch res := val.(type) {
		case map[string]any:
			entry := Entry(res)
			return entry.Get(key[1:]...)
		case Entry:
			return res.Get(key[1:]...)
		default:
			return val, ok
		}
	}
	return nil, false
}

func (d Entry) Set(value any, key ...string) {
	if len(key) == 0 {
		return
	}
	if len(key) == 1 {
		d[key[0]] = value
		return
	}
	if val, ok := d[key[0]]; ok {
		switch res := val.(type) {
		case map[string]any:
			// recurse with key remainder
			entry := Entry(res)
			entry.Set(value, key[1:]...)
		case Entry:
			res.Set(value, key[1:]...)
		default:
			// key exists but is not a map, create new map
			d[key[0]] = make(Entry)
			entry := d[key[0]].(Entry)
			entry.Set(value, key[1:]...)
		}
	} else {
		// key does not exist, create new map
		d[key[0]] = make(Entry)
		entry := d[key[0]].(Entry)
		entry.Set(value, key[1:]...)
	}
}

func (d Entry) SetWithDotKey(key string, value any) {
	d.Set(value, strings.Split(key, ".")...)
}

func (d Entry) KeyExists(key ...string) bool {
	_, ok := d.Get(key...)
	return ok
}

func (d Entry) KeyExistsWithDotKey(key string) bool {
	return d.KeyExists(strings.Split(key, ".")...)
}

// GetMap is a wrapper to simplify recursive map lookup
// since maps are actually pointers and can thus be nil, we can streamline
// the API by omitting the boolean OK. User needs to perform nil check instead
func (d Entry) GetMap(key ...string) Entry {
	if val, ok := d.Get(key...); ok {
		switch t := val.(type) {
		case map[string]any:
			return t
		case Entry:
			return t
		}
	}
	return nil
}

// GetString wraps Get to cast item to string. Returns 3-tuple with value, key presence and
// correct type assertion respectively.
func (d Entry) GetString(key ...string) (string, bool) {
	if val, ok := d.Get(key...); ok {
		switch t := val.(type) {
		case int:
			return strconv.Itoa(t), true
		default:
			str, ok := val.(string)
			if !ok {
				return "", false
			}
			return str, true
		}
	}
	return "", false
}

func (d Entry) GetStringWithDotKey(key string) (string, bool) {
	return d.GetString(strings.Split(key, ".")...)
}

// AssertGetString wraps GetString but omits second truth return value, assuming user knows the type already
// simply returns empty string if GetString type cast failed, omitting the error from user
func (d Entry) AssertGetString(key ...string) string {
	val, _ := d.GetString(key...)
	return val
}

// GetBool is a helper to fetch a boolean value
func (d Entry) GetBool(key ...string) bool {
	if val, ok := d.Get(key...); ok {
		if v, ok := val.(bool); ok {
			return v
		}
	}
	return false
}

func (d Entry) GetInt(key ...string) (int, bool) {
	if val, ok := d.Get(key...); ok {
		switch t := val.(type) {
		case int:
			return t, true
		case float64:
			return int(t), true
		}
	}
	return 0, false
}

func (d Entry) AssertGetInt(key ...string) int {
	val, _ := d.GetInt(key...)
	return val
}

// GetAddr attempts to fetch a string value and parse it as IP address, wrapping the error handling for more concise calls
func (d Entry) GetAddr(key ...string) (netip.Addr, bool) {
	str, ok := d.GetString(key...)
	if !ok {
		return netip.Addr{}, ok
	}
	addr, err := netip.ParseAddr(str)
	if err != nil {
		return addr, false
	}
	return addr, true
}

// GetTimestamp attempts to fetch a string and parse it as timestamp, wrapping error handling for more concise calls
func (d Entry) GetTimestamp(format string, key ...string) (time.Time, bool) {
	str, ok := d.GetString(key...)
	if !ok {
		return time.Time{}, false
	}
	ts, err := time.Parse(format, str)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

// GetStringSlice retrieves nested key as slice of strings if applicable
// for example, we may need to retrieve flowbits from records
// it assumes bad behavior where list can have multiple types (all too common with python output)
// so it also functions as string item extractor, only returning list elements where
// type conversion succeeds. Direct conversion to []string cannot be done with
// map[string]interface decode. Decoded slice is []interface{} and need to be recast per element
func (d Entry) GetStringSlice(key ...string) ([]string, bool) {
	if val, ok := d.Get(key...); ok {
		switch t := val.(type) {
		case []string:
			return t, true
		default:
			slc, ok := val.([]any)
			if !ok {
				return nil, false
			}
			// empty slice with max capacity, as not all []interface elements might be strings
			// capacity ensures only one allocation is done by append
			result := make([]string, 0, len(slc))
			for _, val := range slc {
				if s, ok := val.(string); ok {
					result = append(result, s)
				}
			}
			if len(result) == 0 {
				return result, false
			}
			return result, true
		}
	}
	return nil, false
}

// AssertGetStringSlice omits truth value from GetStringSlice for more concise calls
func (d Entry) AssertGetStringSlice(key ...string) []string {
	v, _ := d.GetStringSlice(key...)
	return v
}

func (d Entry) GetAnySlice(key ...string) []any {
	val, ok := d.Get(key...)
	if !ok {
		return nil
	}
	t, ok := val.([]any)
	if !ok {
		return nil
	}
	return t
}

// GetMapFromSliceByIdx retrieves a specific element from a slice of maps
// wraps a lot of typecaseting boilerplate
// often we need only the first element of the slice, for example domain filter fetches first rrname from dns.query if event_type is alert
func (d Entry) GetMapFromSliceByIdx(idx int, key ...string) Entry {
	if s := d.GetAnySlice(key...); s != nil && len(s) >= idx+1 {
		if t, ok := s[idx].(map[string]any); ok {
			return t
		}
	}
	return nil
}

// GetSliceOfMapsIterator is for processing lists of maps
// usually when decoded, each element needs to be typecase into a map, which is a lot of boilerplate
// this helper accesses a key and casts into slice of any values
// if succeedsful, it returns a channel that allows user to iterate over individual elements as DynamicEntries
func (d Entry) GetSliceOfMapsIterator(key ...string) <-chan Entry {
	slc := d.GetAnySlice(key...)
	if slc == nil {
		return nil
	}
	tx := make(chan Entry)
	go func() {
		defer close(tx)
		for _, v := range slc {
			if t, ok := v.(map[string]any); ok {
				tx <- t
			}
		}
	}()
	return tx
}

func (d Entry) Copy() Entry {
	d2 := make(Entry)
	for k, v := range d {
		switch vt := v.(type) {
		case Entry:
			d2[k] = vt.Copy()
		case map[string]any:
			d2[k] = Entry(vt).Copy()
		case []any:
			dst := make([]any, 0, len(vt))
			d2[k] = append(dst, vt...)
		case *any:
			dup := *vt
			d2[k] = &dup
		default:
			d2[k] = vt
		}
	}
	return d2
}

// Recursive merge of two DynamicEntry objects
func (d Entry) Merge(other Entry) {
	for k, v := range other {
		if val, ok := d[k]; ok {
			switch vt := val.(type) {
			case Entry:
				if ov, ok := v.(Entry); ok {
					vt.Merge(ov)
				}
			case map[string]any:
				if ov, ok := v.(map[string]any); ok {
					maps.Copy(vt, ov)
				}
			default:
				d[k] = v
			}
		} else {
			d[k] = v
		}
	}
}

func (d Entry) Keys(sorted bool) []string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	if sorted {
		sort.Strings(keys)
	}
	return keys
}

// KeysRecurse builds a recursive list of keys with dot notation
func (d Entry) KeysRecurse(sorted bool) []string {
	keys := make([]string, 0, len(d))
	for k, v := range d {
		switch t := v.(type) {
		case Entry:
			keys = append(keys, concatKeys(k, t.KeysRecurse(sorted))...)
		case map[string]any:
			keys = append(keys, concatKeys(k, Entry(t).KeysRecurse(sorted))...)
		default:
			keys = append(keys, k)
		}
	}
	if sorted {
		sort.Strings(keys)
	}
	return keys
}

func concatKeys(base string, sub []string) []string {
	tx := make([]string, 0, len(sub))
	for _, v := range sub {
		tx = append(tx, base+"."+v)
	}
	return tx
}
