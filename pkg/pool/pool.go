// Package pool provides a generic object pool for values whose pointer type
// implements a Reset() method.
package pool

import "sync"

// Resetter describes types that can be reset to a clean state.
type Resetter interface {
	Reset()
}

type pointerToResetter[T any] interface {
	*T
	Resetter
}

// Pool is a generic reusable object pool.
//
// PT must be *T, and *T must implement Resetter. Get returns an object
// from the pool or calls New to create a new one. Put resets the object
// and returns it to the pool.
//
//	p := &pool.Pool[MyStruct, *MyStruct]{
//	    New: func() *MyStruct { return &MyStruct{} },
//	}
type Pool[T any, PT pointerToResetter[T]] struct {
	New  func() PT
	once sync.Once
	pool sync.Pool
}

func (p *Pool[T, PT]) init() {
	p.once.Do(
		func() {
			newFn := p.New
			p.pool.New = func() any { return newFn() }
		},
	)
}

// Get returns an object from the pool. If the pool is empty, it calls New
// to create a new object.
func (p *Pool[T, PT]) Get() PT {
	p.init()
	return p.pool.Get().(PT)
}

// Put resets v and returns it to the pool.
func (p *Pool[T, PT]) Put(v PT) {
	p.init()
	v.Reset()
	p.pool.Put(v)
}
