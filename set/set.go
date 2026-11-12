package set

type Set[T comparable] interface {
	Add(T)
	Has(T) bool
	Delete(T) bool
	Slice() []T
	Load(...T) int
	Len() int
}
