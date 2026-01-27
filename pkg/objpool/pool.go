package objpool

import "sync"

// Resettable интерфейс для объектов с Reset
type Resettable interface {
	Reset()
}

// Pool - generic пул объектов
type Pool[T Resettable] struct {
	internal sync.Pool
}

// New создает пул
func New[T Resettable](newFunc func() T) *Pool[T] {
	return &Pool[T]{internal: sync.Pool{
		New: func() interface{} { return newFunc() },
	}}
}

func (p *Pool[T]) Get() T {
	return p.internal.Get().(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.internal.Put(obj)
}
