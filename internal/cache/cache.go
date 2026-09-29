// Package cache holds the two small in-memory caches the aggregator needs.
package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Value caches the result of an expensive fetch for ttl. Concurrent misses
// share one fetch, and a failed refresh serves the last good value rather than
// an error.
type Value[T any] struct {
	ttl   time.Duration
	fetch func(context.Context) (T, error)

	mu sync.RWMutex
	v  T
	at time.Time
	ok bool
	sf singleflight.Group
}

func NewValue[T any](ttl time.Duration, fetch func(context.Context) (T, error)) *Value[T] {
	return &Value[T]{ttl: ttl, fetch: fetch}
}

func (c *Value[T]) fresh() (T, bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v, c.ok, c.ok && time.Since(c.at) < c.ttl
}

// Get returns the cached value, fetching it first when it is missing or older
// than the TTL.
func (c *Value[T]) Get(ctx context.Context) (T, error) {
	if v, _, fresh := c.fresh(); fresh {
		return v, nil
	}
	v, err, _ := c.sf.Do("", func() (any, error) {
		if v, _, fresh := c.fresh(); fresh {
			return v, nil // refreshed while we waited
		}
		v, err := c.fetch(ctx)
		if err != nil {
			return v, err
		}
		c.mu.Lock()
		c.v, c.at, c.ok = v, time.Now(), true
		c.mu.Unlock()
		return v, nil
	})
	if err != nil {
		if stale, ok, _ := c.fresh(); ok {
			return stale, nil
		}
		var zero T
		return zero, err
	}
	return v.(T), nil
}

// Invalidate forces the next Get to fetch.
func (c *Value[T]) Invalidate() {
	c.mu.Lock()
	c.ok = false
	c.mu.Unlock()
}

// Map is a small keyed TTL cache. It is cleared outright when it outgrows
// maxEntries: entries are cheap to rebuild and the key space is tiny, so
// precise eviction is not worth the code.
type Map[K comparable, V any] struct {
	ttl        time.Duration
	maxEntries int

	mu sync.Mutex
	m  map[K]entry[V]
}

type entry[V any] struct {
	v  V
	at time.Time
}

func NewMap[K comparable, V any](ttl time.Duration, maxEntries int) *Map[K, V] {
	return &Map[K, V]{ttl: ttl, maxEntries: maxEntries, m: map[K]entry[V]{}}
}

// GetOrLoad returns the cached value for k, loading and storing it on a miss.
func (c *Map[K, V]) GetOrLoad(k K, load func() (V, error)) (V, error) {
	c.mu.Lock()
	e, ok := c.m[k]
	c.mu.Unlock()
	if ok && time.Since(e.at) < c.ttl {
		return e.v, nil
	}
	v, err := load()
	if err != nil {
		return v, err
	}
	c.mu.Lock()
	if len(c.m) >= c.maxEntries {
		clear(c.m)
	}
	c.m[k] = entry[V]{v, time.Now()}
	c.mu.Unlock()
	return v, nil
}
