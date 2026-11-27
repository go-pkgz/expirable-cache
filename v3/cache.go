// Package cache implements Cache similar to hashicorp/golang-lru
//
// Support LRC, LRU and TTL-based eviction.
// Package is thread-safe and doesn't spawn any goroutines.
// Callbacks set with WithOnEvicted and the predicate of InvalidateFn run with the cache lock held
// and must not call the cache, or they deadlock.
// When a new key is added (Set, Add, ContainsOrAdd), the cache removes the oldest entry if it has expired,
// unless both the cache and the new entry use the default TTL; updating an existing key removes nothing.
// In case WithMaxKeys is set, cache deletes the oldest entry disregarding its expiration date to maintain the size,
// either using LRC or LRU eviction.
// In case of default TTL (10 years) and default WithMaxKeys (0, unlimited) the cache will be truly unlimited
// and will never delete entries from itself automatically, unless entries are given their own TTL.
//
// Get, Peek, Contains and Values treat an expired entry as missing, while Len and Keys report
// everything the cache still holds, expired entries included.
//
// Important: only reliable way of not having expired entries stuck in a cache is to
// run cache.DeleteExpired periodically using time.Ticker, advisable period is 1/2 of TTL.
package cache

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// Cache defines cache interface
type Cache[K comparable, V any] interface {
	fmt.Stringer

	// WithTTL sets the cache-wide TTL, used by Add, ContainsOrAdd and Set with a ttl of 0, for entries
	// added or updated afterwards. By default, it is set to 10 years, which is the way to disable expiry;
	// sane option for expirable cache might be 5 minutes. A TTL of 0 or less makes those entries expire at once.
	WithTTL(ttl time.Duration) Cache[K, V]
	// WithMaxKeys functional option defines how many keys to keep.
	// By default, it is 0, which means unlimited.
	WithMaxKeys(maxKeys int) Cache[K, V]
	// WithLRU sets cache to LRU (Least Recently Used) eviction mode, in which Get also moves an entry to the front.
	// By default, the mode is LRC (Least Recently Created), in which only adding or updating a key does.
	WithLRU() Cache[K, V]
	// WithOnEvicted defines a function called for every removed entry, whether removed automatically or manually.
	// It runs with the cache lock held and must not call the cache, or it deadlocks.
	WithOnEvicted(fn func(key K, value V)) Cache[K, V]

	// Add adds a value to the cache with the cache-wide TTL, or replaces the value and the TTL of an existing key.
	// Returns true if an eviction occurred, either because the size was exceeded or because the oldest entry expired.
	// Returns false if there was no eviction: the item was already in the cache, or nothing had to be removed.
	Add(key K, value V) bool
	// Set adds a value with its own TTL, or replaces the value and the TTL of an existing key.
	// A ttl of 0 uses the cache-wide TTL; a negative ttl makes the entry expire at once.
	Set(key K, value V, ttl time.Duration)
	// Get returns the key value and whether it is found and not expired.
	// An expired entry returns its stored value with false.
	Get(key K) (V, bool)
	// GetExpiration returns the expiration time of the key. Non-existing key returns zero time.
	GetExpiration(key K) (time.Time, bool)
	// GetOldest returns the oldest entry
	GetOldest() (K, V, bool)
	// Contains checks if a key is in the cache, without updating the recent-ness
	// or deleting it for being stale. Expired entries are reported as missing.
	Contains(key K) (ok bool)
	// ContainsOrAdd checks if a key is in the cache and adds the value if it is not, under one lock.
	// A found key changes nothing: no recent-ness update and no stats. An expired entry counts as
	// missing and is replaced the same way Add updates an existing key.
	// Returns whether the key was found, and whether adding it caused an eviction.
	ContainsOrAdd(key K, value V) (ok, evicted bool)
	// Peek returns the key value (or undefined if not found) without updating the "recently used"-ness of the key.
	// Works exactly the same as Get in case of LRC mode (default one).
	Peek(key K) (V, bool)
	// Values returns a slice of the values in the cache, from oldest to newest.
	// Expired entries are filtered out.
	Values() []V
	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K
	// Len return count of items in cache, including expired
	Len() int
	// Remove removes the provided key from the cache, returning if the
	// key was contained.
	Remove(key K) bool
	// Invalidate key (item) from the cache
	Invalidate(key K)
	// InvalidateFn deletes multiple keys if predicate is true.
	// The predicate runs with the cache lock held and must not call the cache, or it deadlocks.
	InvalidateFn(fn func(key K) bool)
	// RemoveOldest remove the oldest element in the cache
	RemoveOldest() (K, V, bool)
	// DeleteExpired clears cache of expired items
	DeleteExpired()
	// Purge clears the cache completely.
	Purge()
	// Resize changes the cache size and returns the number of entries it removed.
	// Size of 0 or less means unlimited.
	Resize(int) int
	// Stat gets the current stats for cache
	Stat() Stats
}

// Stats provides statistics for cache.
// Hits and Misses count lookups by Get and Peek; an expired entry counts as a miss.
// Added counts new keys, not updates of existing ones.
// Evicted counts every entry removed for any reason, including explicit removal and Purge.
type Stats struct {
	Hits, Misses   int // cache effectiveness
	Added, Evicted int // number of added and evicted records
}

// cacheImpl provides Cache interface implementation.
type cacheImpl[K comparable, V any] struct {
	ttl       time.Duration
	maxKeys   int
	isLRU     bool
	onEvicted func(key K, value V)

	sync.Mutex
	stat      Stats
	items     map[K]*list.Element
	evictList *list.List
}

// noEvictionTTL - very long ttl to prevent eviction
const noEvictionTTL = time.Hour * 24 * 365 * 10

// NewCache returns a new Cache.
// Default MaxKeys is unlimited (0).
// Default TTL is 10 years, sane value for expirable cache is 5 minutes.
// Default eviction mode is LRC, appropriate option allow to change it to LRU.
func NewCache[K comparable, V any]() Cache[K, V] {
	return &cacheImpl[K, V]{
		items:     map[K]*list.Element{},
		evictList: list.New(),
		ttl:       noEvictionTTL,
		maxKeys:   0,
	}
}

func (c *cacheImpl[K, V]) Add(key K, value V) (evicted bool) {
	return c.addWithTTL(key, value, 0)
}

func (c *cacheImpl[K, V]) Set(key K, value V, ttl time.Duration) {
	c.addWithTTL(key, value, ttl)
}

// Returns true if an eviction occurred, either because the size was exceeded
// or because the oldest entry expired.
// Returns false if there was no eviction: the item was already in the cache,
// or nothing had to be removed.
func (c *cacheImpl[K, V]) addWithTTL(key K, value V, ttl time.Duration) (evicted bool) {
	now := time.Now()
	c.Lock()
	defer c.Unlock()
	if ttl == 0 {
		ttl = c.ttl
	}
	return c.add(key, value, ttl, now)
}

// add implements addWithTTL. Has to be called with lock!
func (c *cacheImpl[K, V]) add(key K, value V, ttl time.Duration, now time.Time) (evicted bool) {
	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*cacheItem[K, V]).value = value
		ent.Value.(*cacheItem[K, V]).expiresAt = now.Add(ttl)
		return false
	}

	// Add new item
	ent := &cacheItem[K, V]{key: key, value: value, expiresAt: now.Add(ttl)}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry
	c.stat.Added++

	// Remove the oldest entry if it is expired, only in case of non-default TTL.
	if c.ttl != noEvictionTTL || ttl != noEvictionTTL {
		ent := c.evictList.Back()
		if ent != nil && now.After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			c.removeElement(ent)
			evicted = true
		}
	}

	// Verify size not exceeded
	if c.maxKeys > 0 && len(c.items) > c.maxKeys {
		c.removeOldest()
		evicted = true
	}
	return evicted
}

func (c *cacheImpl[K, V]) Get(key K) (V, bool) {
	def := *new(V)
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			c.stat.Misses++
			return ent.Value.(*cacheItem[K, V]).value, false
		}
		if c.isLRU {
			c.evictList.MoveToFront(ent)
		}
		c.stat.Hits++
		return ent.Value.(*cacheItem[K, V]).value, true
	}
	c.stat.Misses++
	return def, false
}

func (c *cacheImpl[K, V]) Contains(key K) (ok bool) {
	c.Lock()
	defer c.Unlock()
	ent, ok := c.items[key]
	if !ok {
		return false
	}
	return !time.Now().After(ent.Value.(*cacheItem[K, V]).expiresAt)
}

func (c *cacheImpl[K, V]) ContainsOrAdd(key K, value V) (ok, evicted bool) {
	c.Lock()
	defer c.Unlock()
	now := time.Now()
	if ent, found := c.items[key]; found && !now.After(ent.Value.(*cacheItem[K, V]).expiresAt) {
		return true, false
	}
	return false, c.add(key, value, c.ttl, now)
}

func (c *cacheImpl[K, V]) Peek(key K) (V, bool) {
	def := *new(V)
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		// Expired item check
		if time.Now().After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			c.stat.Misses++
			return ent.Value.(*cacheItem[K, V]).value, false
		}
		c.stat.Hits++
		return ent.Value.(*cacheItem[K, V]).value, true
	}
	c.stat.Misses++
	return def, false
}

func (c *cacheImpl[K, V]) GetExpiration(key K) (time.Time, bool) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		return ent.Value.(*cacheItem[K, V]).expiresAt, true
	}
	return time.Time{}, false
}

func (c *cacheImpl[K, V]) Keys() []K {
	c.Lock()
	defer c.Unlock()
	return c.keys()
}

func (c *cacheImpl[K, V]) Values() []V {
	now := time.Now()
	c.Lock()
	defer c.Unlock()
	values := make([]V, 0, len(c.items))
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		if !now.After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			values = append(values, ent.Value.(*cacheItem[K, V]).value)
		}
	}
	return values
}

func (c *cacheImpl[K, V]) Len() int {
	c.Lock()
	defer c.Unlock()
	return c.evictList.Len()
}

func (c *cacheImpl[K, V]) Resize(size int) int {
	c.Lock()
	defer c.Unlock()
	if size <= 0 {
		c.maxKeys = 0
		return 0
	}
	diff := max(c.evictList.Len()-size, 0)
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.maxKeys = size
	return diff
}

func (c *cacheImpl[K, V]) Invalidate(key K) {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}

func (c *cacheImpl[K, V]) InvalidateFn(fn func(key K) bool) {
	c.Lock()
	defer c.Unlock()
	for key, ent := range c.items {
		if fn(key) {
			c.removeElement(ent)
		}
	}
}

func (c *cacheImpl[K, V]) Remove(key K) bool {
	c.Lock()
	defer c.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

func (c *cacheImpl[K, V]) RemoveOldest() (key K, value V, ok bool) {
	c.Lock()
	defer c.Unlock()
	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent)
		return ent.Value.(*cacheItem[K, V]).key, ent.Value.(*cacheItem[K, V]).value, true
	}
	return
}

func (c *cacheImpl[K, V]) GetOldest() (key K, value V, ok bool) {
	c.Lock()
	defer c.Unlock()
	if ent := c.evictList.Back(); ent != nil {
		return ent.Value.(*cacheItem[K, V]).key, ent.Value.(*cacheItem[K, V]).value, true
	}
	return
}

func (c *cacheImpl[K, V]) DeleteExpired() {
	now := time.Now()
	c.Lock()
	defer c.Unlock()
	var nextEnt *list.Element
	for ent := c.evictList.Back(); ent != nil; ent = nextEnt {
		nextEnt = ent.Prev()
		if now.After(ent.Value.(*cacheItem[K, V]).expiresAt) {
			c.removeElement(ent)
		}
	}
}

func (c *cacheImpl[K, V]) Purge() {
	c.Lock()
	defer c.Unlock()
	for k, v := range c.items {
		delete(c.items, k)
		c.stat.Evicted++
		if c.onEvicted != nil {
			c.onEvicted(k, v.Value.(*cacheItem[K, V]).value)
		}
	}
	c.evictList.Init()
}

func (c *cacheImpl[K, V]) Stat() Stats {
	c.Lock()
	defer c.Unlock()
	return c.stat
}

func (c *cacheImpl[K, V]) String() string {
	stats := c.Stat()
	size := c.Len()
	ratio := 0.0
	if total := stats.Hits + stats.Misses; total > 0 {
		ratio = 100 * float64(stats.Hits) / float64(total)
	}
	return fmt.Sprintf("Size: %d, Stats: %+v (%0.1f%%)", size, stats, ratio)
}

// Keys returns a slice of the keys in the cache, from oldest to newest. Has to be called with lock!
func (c *cacheImpl[K, V]) keys() []K {
	keys := make([]K, 0, len(c.items))
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys = append(keys, ent.Value.(*cacheItem[K, V]).key)
	}
	return keys
}

// removeOldest removes the oldest item from the cache. Has to be called with lock!
func (c *cacheImpl[K, V]) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache. Has to be called with lock!
func (c *cacheImpl[K, V]) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*cacheItem[K, V])
	delete(c.items, kv.key)
	c.stat.Evicted++
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

// cacheItem is used to hold a value in the evictList
type cacheItem[K comparable, V any] struct {
	expiresAt time.Time
	key       K
	value     V
}
