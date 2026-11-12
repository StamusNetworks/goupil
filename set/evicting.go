package set

import (
	lru "github.com/hashicorp/golang-lru/v2"
)

type Evicting[T comparable] struct {
	data *lru.TwoQueueCache[T, bool]
}

func (s *Evicting[T]) Add(t T) {
	s.data.Add(t, true)
}

func (s *Evicting[T]) Has(t T) bool {
	return s.data.Contains(t)
}

func (s *Evicting[T]) Delete(t T) (had bool) {
	had = s.Has(t)
	if had {
		s.data.Remove(t)
	}
	return had
}

func (s *Evicting[T]) Slice() []T {
	return s.data.Keys()
}

func (s *Evicting[T]) Load(items ...T) int {
	for _, item := range items {
		s.Add(item)
	}
	return len(items)
}

func (s *Evicting[T]) Len() int {
	return s.data.Len()
}

func NewEvicting[T comparable](size int) (*Evicting[T], error) {
	cache, err := lru.New2Q[T, bool](size)
	if err != nil {
		return nil, err
	}
	return &Evicting[T]{data: cache}, nil
}
