// Package pool provides a generic fixed-size object pool for values whose
// pointer type implements a Reset() method.
package pool

import "fmt"

// MaxSize is the maximum number of objects a Pool may hold.
const MaxSize = 1000

// Resetter describes types that can be reset to a clean state.
type Resetter interface {
	Reset()
}

type pointerToResetter[T any] interface {
	*T
	Resetter
}

// Pool is a fixed-size pool of reusable pointers to values of type T.
//
// PT must be *T, and *T must implement Resetter. Get blocks when the pool
// is empty; Put blocks when the pool is full.
//
//	Pool for type MyStruct and pointer *MyStruct:
//	p := pool.New[MyStruct, *MyStruct](size)
type Pool[T any, PT pointerToResetter[T]] struct {
	ch chan PT
}

// New creates a fixed-size Pool containing size pre-allocated objects.
// It returns an error if size is not between 1 and MaxSize.
func New[T any, PT pointerToResetter[T]](size int) (*Pool[T, PT], error) {
	if size <= 0 || size > MaxSize {
		return nil, fmt.Errorf("pool.New: size must be between 1 and %d", MaxSize)
	}

	ch := make(chan PT, size)
	for range size {
		var t T
		ch <- PT(&t)
	}

	return &Pool[T, PT]{ch: ch}, nil
}

// Get returns an object from the pool. It blocks until an object is available.
func (p *Pool[T, PT]) Get() PT {
	return <-p.ch
}

// Put resets v and returns it to the pool. It blocks if the pool is full.
func (p *Pool[T, PT]) Put(v PT) {
	v.Reset()
	p.ch <- v
}
