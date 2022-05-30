package cache

import "time"

type options[K comparable, V any] struct {
	ttl       *time.Duration
	maxKeys   *int
	lru       *bool
	onEvicted func(key K, value V)
}

// Options is a function that created a new empty option.
func Options[K comparable, V any]() *options[K, V] { return &options[K, V]{} } // nolint

// TTL functional option defines TTL for all cache entries.
// By default, it is set to 10 years, sane option for expirable cache might be 5 minutes.
func (o *options[K, V]) TTL(ttl time.Duration) *options[K, V] {
	o.ttl = &ttl
	return o
}

// MaxKeys functional option defines how many keys to keep.
// By default, it is 0, which means unlimited.
func (o *options[K, V]) MaxKeys(maxKeys int) *options[K, V] {
	o.maxKeys = &maxKeys
	return o
}

// LRU sets cache to LRU (Least Recently Used) eviction mode.
func (o *options[K, V]) LRU() *options[K, V] {
	v := true
	o.lru = &v
	return o
}

// OnEvicted called automatically for automatically and manually deleted entries
func (o *options[K, V]) OnEvicted(fn func(key K, value V)) *options[K, V] {
	o.onEvicted = fn
	return o
}
