// Package cache implements Cache similar to hashicorp/golang-lru
//
// Support LRC, LRU and TTL-based eviction.
// Package is thread-safe and doesn't spawn any goroutines.
// On every Set() call, cache deletes single oldest entry in case it's expired.
// In case MaxSize is set, cache deletes the oldest entry disregarding its expiration date to maintain the size,
// either using LRC or LRU eviction.
// In case of default TTL (10 years) and default MaxSize (0, unlimited) the cache will be truly unlimited
// and will never delete entries from itself automatically.
//
// Important: only reliable way of not having expired entries stuck in a cache is to
// run cache.DeleteExpired periodically using time.Ticker, advisable period is 1/2 of TTL.
package cache

import (
	"fmt"
	"time"

	v3 "github.com/go-pkgz/expirable-cache/v3"
)

// Cache defines cache interface
type Cache[K comparable, V any] interface {
	fmt.Stringer
	options[K, V]
	Set(key K, value V, ttl time.Duration)
	Get(key K) (V, bool)
	Peek(key K) (V, bool)
	Keys() []K
	Len() int
	Invalidate(key K)
	InvalidateFn(fn func(key K) bool)
	RemoveOldest()
	DeleteExpired()
	Purge()
	Stat() Stats
}

// Stats provides statistics for cache
type Stats struct {
	Hits, Misses   int // cache effectiveness
	Added, Evicted int // number of added and evicted records
}

// cacheImpl provides Cache interface implementation.
type cacheImpl[K comparable, V any] struct {
	v3.Cache[K, V]
}

// NewCache returns a new Cache.
// Default MaxKeys is unlimited (0).
// Default TTL is 10 years, sane value for expirable cache is 5 minutes.
// Default eviction mode is LRC, appropriate option allow to change it to LRU.
func NewCache[K comparable, V any]() Cache[K, V] {
	return &cacheImpl[K, V]{v3.NewCache[K, V]()}
}

func (c *cacheImpl[K, V]) Get(key K) (V, bool) {
	value, ok := c.Cache.Get(key)
	if !ok {
		// preserve v2 behavior of not returning value in case it's expired
		// which is not compatible with v3 and simplelru
		def := *new(V)
		return def, ok
	}
	return value, ok
}

func (c *cacheImpl[K, V]) Peek(key K) (V, bool) {
	value, ok := c.Cache.Peek(key)
	if !ok {
		// preserve v2 behavior of not returning value in case it's expired
		// which is not compatible with v3 and simplelru
		def := *new(V)
		return def, ok
	}
	return value, ok
}

func (c *cacheImpl[K, V]) RemoveOldest() {
	c.Cache.RemoveOldest()
}

func (c *cacheImpl[K, V]) Stat() Stats {
	stats := c.Cache.Stat()
	return Stats{
		Hits:    stats.Hits,
		Misses:  stats.Misses,
		Added:   stats.Added,
		Evicted: stats.Evicted,
	}
}
