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

func TestPool_NewReturnsErrorOnInvalidSize(t *testing.T) {
	_, err := pool.New[item, *item](0)
	require.Error(t, err)

	_, err = pool.New[item, *item](-1)
	require.Error(t, err)

	_, err = pool.New[item, *item](pool.MaxSize + 1)
	require.Error(t, err)
}

func TestPool_GetReturnsFreshObject(t *testing.T) {
	p, err := pool.New[item, *item](10)
	require.NoError(t, err)
	require.NotNil(t, p)

	obj := p.Get()
	require.NotNil(t, obj)

	assert.Zero(t, obj.value)
	assert.False(t, obj.seen)
}

func TestPool_PutResetsObject(t *testing.T) {
	p, err := pool.New[item, *item](10)
	require.NoError(t, err)

	obj := p.Get()
	obj.value = 42
	obj.seen = true

	p.Put(obj)

	obj2 := p.Get()
	assert.Equal(t, 0, obj2.value)
	assert.False(t, obj2.seen)
}

func TestPool_ReusesObjects(t *testing.T) {
	p, err := pool.New[item, *item](1)
	require.NoError(t, err)

	obj := p.Get()
	ptr := obj

	p.Put(obj)

	obj2 := p.Get()
	assert.Same(t, ptr, obj2)
}

func TestPool_MultipleGetPut(t *testing.T) {
	p, err := pool.New[item, *item](10)
	require.NoError(t, err)

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
	p, err := pool.New[item, *item](100)
	require.NoError(t, err)

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
