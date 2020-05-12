// Package cache implements LoadingCache similar to hashicorp/golang-lru
//
// Support LRC, LRU and TTL-based eviction.
// Package is thread-safe and doesn't spawn any goroutines.
// On every Set() call, cache deletes single oldest entry in cache in case it's expired.
// In case MaxSize is set, cache deletes oldest entry disregarding it's expiration date to maintain the size.
//
// Important: only reliable way of not having expired entries stuck in cache is to
// run DeleteExpired by time.Ticker, advisable time is 1/2 of TTL.
//
// In case of default TTL (10 years) and default MaxSize (0, unlimited) the cache will be truly unlimited
// and will never delete entries from itself automatically.
package cache

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// LoadingCache defines loading cache interface
type LoadingCache interface {
	fmt.Stringer
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Peek(key string) (interface{}, bool)
	Keys() []string
	Len() int
	Invalidate(key string)
	InvalidateFn(fn func(key string) bool)
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

// loadingCacheImpl provides loading cache, implements cache.LoadingCache.
type loadingCacheImpl struct {
	ttl       time.Duration
	maxKeys   int
	isLRU     bool
	onEvicted func(key string, value interface{})

	sync.Mutex
	stat      Stats
	items     map[string]*list.Element
	evictList *list.List
}

// noEvictionTTL - very long ttl to prevent eviction
const noEvictionTTL = time.Hour * 24 * 365 * 10

// NewLoadingCache returns a new cache.
// Default MaxKeys is unlimited (0).
// Default TTL is 10 years, sane value for expirable cache is 5 minutes.
// Default eviction mode is LRC, appropriate option allow to change it to LRU.
func NewLoadingCache(options ...Option) (LoadingCache, error) {
	res := loadingCacheImpl{
		items:     map[string]*list.Element{},
		evictList: list.New(),
		ttl:       noEvictionTTL,
		maxKeys:   0,
	}

	for _, opt := range options {
		if err := opt(&res); err != nil {
			return nil, errors.Wrap(err, "failed to set cache option")
		}
	}
	return &res, nil
}

// Set key
func (c *loadingCacheImpl) Set(key string, value interface{}) {
	c.Lock()
	defer c.Unlock()
	now := time.Now()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*cacheItem).value = value
		ent.Value.(*cacheItem).expiresAt = now.Add(c.ttl)
		return
	}

	// Add new item
	ent := &cacheItem{key: key, value: value, expiresAt: now.Add(c.ttl)}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry
	c.stat.Added++

	// Remove oldest entry if it is expired, only in case of non-default TTL.
	if c.ttl != noEvictionTTL {
		c.removeOldestIfExpired()
	}

	// Verify size not exceeded
	if c.maxKeys > 0 && len(c.items) > c.maxKeys {
		c.removeOldest()
		return
	}
}

// Get returns the key value if it's not expired
func (c *loadingCacheImpl) Get(key string) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.Value.(*cacheItem).expiresAt) {
			c.stat.Misses++
			return nil, false
		}
		if c.isLRU {
			c.evictList.MoveToFront(ent)
		}
		c.stat.Hits++
		return ent.Value.(*cacheItem).value, true
	}
	c.stat.Misses++
	return nil, false
}

// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
func (c *loadingCacheImpl) Peek(key string) (interface{}, bool) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.Value.(*cacheItem).expiresAt) {
			return nil, false
		}
		return ent.Value.(*cacheItem).value, true
	}
	return nil, false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *loadingCacheImpl) Keys() []string {
	c.Lock()
	defer c.Unlock()
	return c.keys()
}

// Len return count of items in cache, including expired
func (c *loadingCacheImpl) Len() int {
	c.Lock()
	defer c.Unlock()
	return c.evictList.Len()
}

// Invalidate key (item) from the cache
func (c *loadingCacheImpl) Invalidate(key string) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}

// InvalidateFn deletes multiple keys if predicate is true
func (c *loadingCacheImpl) InvalidateFn(fn func(key string) bool) {
	c.Lock()
	defer c.Unlock()
	for key, ent := range c.items {
		if fn(key) {
			c.removeElement(ent)
		}
	}
}

// RemoveOldest remove oldest element in the cache
func (c *loadingCacheImpl) RemoveOldest() {
	c.Lock()
	defer c.Unlock()
	c.removeOldest()
}

// DeleteExpired clears cache of expired items
func (c *loadingCacheImpl) DeleteExpired() {
	c.Lock()
	defer c.Unlock()
	for _, key := range c.keys() {
		if time.Now().After(c.items[key].Value.(*cacheItem).expiresAt) {
			c.removeElement(c.items[key])
			continue
		}
		// if cache is not LRU, keys() are sorted by expiresAt and there are no
		// more expired entries left at this point
		if !c.isLRU {
			return
		}
	}
}

// Purge clears the cache completely.
func (c *loadingCacheImpl) Purge() {
	c.Lock()
	defer c.Unlock()
	for k, v := range c.items {
		delete(c.items, k)
		c.stat.Evicted++
		if c.onEvicted != nil {
			c.onEvicted(k, v.Value.(*cacheItem).value)
		}
	}
	c.evictList.Init()
}

// Stat gets the current stats for cache
func (c *loadingCacheImpl) Stat() Stats {
	c.Lock()
	defer c.Unlock()
	return c.stat
}

func (c *loadingCacheImpl) String() string {
	stats := c.Stat()
	size := c.Len()
	return fmt.Sprintf("Size: %d, Stats: %+v (%0.1f%%)", size, stats, 100*float64(stats.Hits)/float64(stats.Hits+stats.Misses))
}

// Keys returns a slice of the keys in the cache, from oldest to newest. Has to be called with lock!
func (c *loadingCacheImpl) keys() []string {
	keys := make([]string, 0, len(c.items))
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys = append(keys, ent.Value.(*cacheItem).key)
	}
	return keys
}

// removeOldest removes the oldest item from the cache. Has to be called with lock!
func (c *loadingCacheImpl) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeOldest removes the oldest item from the cache in case it's already expired. Has to be called with lock!
func (c *loadingCacheImpl) removeOldestIfExpired() {
	ent := c.evictList.Back()
	if ent != nil && time.Now().After(ent.Value.(*cacheItem).expiresAt) {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache. Has to be called with lock!
func (c *loadingCacheImpl) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*cacheItem)
	delete(c.items, kv.key)
	c.stat.Evicted++
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

// cacheItem is used to hold a value in the evictList
type cacheItem struct {
	expiresAt time.Time
	key       string
	value     interface{}
}
