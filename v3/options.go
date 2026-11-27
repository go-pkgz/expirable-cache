package cache

import "time"

// The With* methods are documented on the Cache interface.

func (c *cacheImpl[K, V]) WithTTL(ttl time.Duration) Cache[K, V] {
	c.Lock()
	defer c.Unlock()
	c.ttl = ttl
	return c
}

func (c *cacheImpl[K, V]) WithMaxKeys(maxKeys int) Cache[K, V] {
	c.Lock()
	defer c.Unlock()
	c.maxKeys = maxKeys
	return c
}

func (c *cacheImpl[K, V]) WithLRU() Cache[K, V] {
	c.Lock()
	defer c.Unlock()
	c.isLRU = true
	return c
}

func (c *cacheImpl[K, V]) WithOnEvicted(fn func(key K, value V)) Cache[K, V] {
	c.Lock()
	defer c.Unlock()
	c.onEvicted = fn
	return c
}
