package set

import (
	"errors"
)

type Sharded[T comparable] struct {
	shards []Set[T]
	count  uint64
	hasher func(T) uint64
}

func (s *Sharded[T]) Add(t T) {
	s.pick(t).Add(t)
}

func (s *Sharded[T]) Has(t T) bool {
	return s.pick(t).Has(t)
}

func (s *Sharded[T]) Delete(t T) (had bool) {
	return s.pick(t).Delete(t)
}

// Load condenses bulk load into a simple to call method
// mostly useful for loading persistence
func (s *Sharded[T]) Load(st ...T) int {
	for _, t := range st {
		s.Add(t)
	}
	return len(st)
}

func (s *Sharded[T]) Slice() []T {
	tx := make([]T, 0)
	for _, shard := range s.shards {
		tx = append(tx, shard.Slice()...)
	}
	return tx
}

func (s *Sharded[T]) Len() (count int) {
	for _, shard := range s.shards {
		count += shard.Len()
	}
	return count
}

func (s *Sharded[T]) Distribution() []int {
	tx := make([]int, 0)
	for _, shard := range s.shards {
		tx = append(tx, shard.Len())
	}
	return tx
}

func (s *Sharded[T]) pick(t T) Set[T] {
	return s.shards[s.hasher(t)%s.count]
}

func NewSharded[T comparable](count int, size int, b func(T) uint64) (*Sharded[T], error) {
	if b == nil {
		return nil, errors.New("sharded set requires balancing function")
	}
	if count < 1 {
		return nil, errors.New("invalid shard count")
	}
	m := &Sharded[T]{
		count:  uint64(count),
		shards: make([]Set[T], 0, count),
		hasher: b,
	}
	for range count {
		if size > 0 {
			s, err := NewEvicting[T](size / count)
			if err != nil {
				return nil, err
			}
			m.shards = append(m.shards, s)
		} else {
			// no max size defined, use unbounded set
			m.shards = append(m.shards, NewSafe[T]())
		}
	}
	return m, nil
}
