package generics

// Generic type with single type parameter
type List[T any] struct {
	data []T
}

// Generic type with multiple type parameters
type Map[K comparable, V any] struct {
	keys   []K
	values []V
}

// Generic type with constraint
type Number[T ~int | ~float64] struct {
	value T
}

// Generic function
func Transform[T, U any](input []T, f func(T) U) []U {
	result := make([]U, len(input))
	for i, v := range input {
		result[i] = f(v)
	}
	return result
}

// Generic method
func (l *List[T]) Add(item T) {
	l.data = append(l.data, item)
}

// Using generic types
var (
	intList   List[int]
	stringMap Map[string, int]
	floatNum  Number[float64]
)

// Generic interface
type Container[T any] interface {
	Add(item T)
	Get() T
}

// Implementation of generic interface
type Box[T any] struct {
	value T
}

func (b *Box[T]) Add(item T) {
	b.value = item
}

func (b *Box[T]) Get() T {
	return b.value
}
