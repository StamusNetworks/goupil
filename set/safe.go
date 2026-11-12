package set

import (
	"sync"
)

type Safe[T comparable] struct {
	sync.RWMutex

	set Set[T]
}

// Add is a thread-safe setter for set, writer must have exclusive access to data as write operations
// are not thread safe
func (c *Safe[T]) Add(item T) {
	c.Lock()
	defer c.Unlock()
	c.set.Add(item)
}

// Has is a thread-safe getter from set, multiple workers can read at once
func (c *Safe[T]) Has(item T) bool {
	c.RLock()
	defer c.RUnlock()
	return c.set.Has(item)
}

// Delete removes item from set, once again demanding exclusive access
func (c *Safe[T]) Delete(item T) bool {
	c.Lock()
	defer c.Unlock()
	return c.set.Delete(item)
}

// Slice is to replicate central repository to lock-free slice of strings, for dumping
func (c *Safe[T]) Slice() []T {
	c.RLock()
	defer c.RUnlock()
	return c.set.Slice()
}

// Load is for bulk-inserting multiple labels within one lock cycle
func (c *Safe[T]) Load(items ...T) int {
	c.Lock()
	defer c.Unlock()
	return c.set.Load(items...)
}

// Len is thread-safe helper to count number of frequent items stored
func (c *Safe[T]) Len() int {
	c.RLock()
	defer c.RUnlock()
	return c.set.Len()
}

// NewSafe is constructor for Safe
func NewSafe[T comparable]() *Safe[T] {
	return &Safe[T]{
		RWMutex: sync.RWMutex{},
		set:     make(Items[T]),
	}
}
