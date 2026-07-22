package pool_test

import (
	"sync"
	"testing"

	"github.com/Dmitriy-Shcheklein/urlshortener/pkg/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type item struct {
	value int
	seen  bool
}

func (i *item) Reset() {
	i.value = 0
	i.seen = false
}

func newPool() *pool.Pool[item, *item] {
	return &pool.Pool[item, *item]{
		New: func() *item { return &item{} },
	}
}

func TestPool_GetReturnsFreshObject(t *testing.T) {
	p := newPool()

	obj := p.Get()
	require.NotNil(t, obj)

	assert.Zero(t, obj.value)
	assert.False(t, obj.seen)
}

func TestPool_PutResetsObject(t *testing.T) {
	p := newPool()

	obj := p.Get()
	obj.value = 42
	obj.seen = true

	p.Put(obj)

	obj2 := p.Get()
	assert.Equal(t, 0, obj2.value)
	assert.False(t, obj2.seen)
}

func TestPool_ReusesObjects(t *testing.T) {
	p := newPool()

	obj := p.Get()
	p.Put(obj)

	obj2 := p.Get()
	assert.Same(t, obj, obj2)
}

func TestPool_MultipleGetPut(t *testing.T) {
	p := newPool()

	const count = 10
	objects := make([]*item, 0, count)

	for range count {
		obj := p.Get()
		obj.value = 1
		objects = append(objects, obj)
	}

	for _, obj := range objects {
		p.Put(obj)
	}

	for range count {
		obj := p.Get()
		assert.Zero(t, obj.value)
	}
}

func TestPool_ConcurrentGetPut(t *testing.T) {
	p := newPool()

	const iterations = 1000

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				obj := p.Get()
				obj.value = 42
				obj.seen = true
				p.Put(obj)
			}
		}()
	}

	wg.Wait()
}

func TestPool_GetCreatesNewWhenEmpty(t *testing.T) {
	p := newPool()

	obj1 := p.Get()
	require.NotNil(t, obj1)

	obj2 := p.Get()
	require.NotNil(t, obj2)

	obj3 := p.Get()
	require.NotNil(t, obj3)
}
