package set

import "math/rand"

type Items[T comparable] map[T]bool

func (s Items[T]) Add(t T)      { s[t] = true }
func (s Items[T]) Has(t T) bool { return s[t] }

func (s Items[T]) Delete(t T) bool {
	if s.Has(t) {
		delete(s, t)
		return true
	}
	return false
}

func (s Items[T]) Slice() []T {
	tx := make([]T, 0, len(s))
	for k := range s {
		tx = append(tx, k)
	}
	return tx
}

func (s Items[T]) Load(t ...T) int {
	for _, item := range t {
		s.Add(item)
	}
	return len(t)
}

func (s Items[T]) Len() int { return len(s) }

func (s Items[T]) Extend(s2 Items[T]) {
	for k := range s2 {
		s.Add(k)
	}
}

func (s Items[T]) SliceNil() []T {
	if len(s) == 0 {
		return nil
	}
	return s.Slice()
}

func (s Items[T]) Top(k int) []T {
	if k < 0 || k > len(s) || len(s) == 0 {
		return s.Slice()
	}
	tx := make([]T, 0, k)
	for item := range s {
		if len(tx) == k {
			break
		}
		tx = append(tx, item)
	}
	return tx
}

func (c Items[T]) Copy() Items[T] {
	dst := make(Items[T])
	for item, val := range c {
		if val {
			dst[item] = true
		}
	}
	return dst
}

func (s Items[T]) DropRandom() {
	idx := rand.Intn(len(s)) //nolint:gosec
	i := 0
	for k := range s {
		if i == idx {
			delete(s, k)
		}
		i++
	}
}

func NewItems[T comparable]() Items[T] {
	return make(Items[T])
}
