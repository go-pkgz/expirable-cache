//go:build !go1.25

package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheWithDeleteExpired(t *testing.T) {
	var evicted []string
	lc := NewCache[string, string]().WithTTL(150 * time.Millisecond).WithOnEvicted(
		func(key string, value string) {
			evicted = append(evicted, key, value)
		})

	lc.Set("key1", "val1", 0)

	time.Sleep(100 * time.Millisecond) // not enough to expire
	lc.DeleteExpired()
	assert.Equal(t, 1, lc.Len())

	v, ok := lc.Get("key1")
	assert.Equal(t, "val1", v)
	assert.True(t, ok)

	time.Sleep(200 * time.Millisecond) // expire
	lc.DeleteExpired()
	v, ok = lc.Get("key1")
	assert.False(t, ok)
	assert.Equal(t, "", v)

	assert.Equal(t, 0, lc.Len())
	assert.Equal(t, []string{"key1", "val1"}, evicted)

	// add new entry
	lc.Set("key2", "val2", 0)
	assert.Equal(t, 1, lc.Len())

	// nothing deleted
	lc.DeleteExpired()
	assert.Equal(t, 1, lc.Len())
	assert.Equal(t, []string{"key1", "val1"}, evicted)

	// Purge, cache should be clean
	lc.Purge()
	assert.Equal(t, 0, lc.Len())
	assert.Equal(t, []string{"key1", "val1", "key2", "val2"}, evicted)
}

func TestCacheExpired(t *testing.T) {
	lc := NewCache[string, string]().WithTTL(time.Millisecond * 5)

	lc.Set("key1", "val1", 0)
	assert.Equal(t, 1, lc.Len())

	v, ok := lc.Peek("key1")
	assert.Equal(t, v, "val1")
	assert.True(t, ok)

	v, ok = lc.Get("key1")
	assert.Equal(t, v, "val1")
	assert.True(t, ok)

	time.Sleep(time.Millisecond * 10) // wait for entry to expire
	assert.Equal(t, 1, lc.Len())      // but not purged

	v, ok = lc.Peek("key1")
	assert.Equal(t, "val1", v, "expired and marked as such, but value is available")
	assert.False(t, ok)

	v, ok = lc.Get("key1")
	assert.Equal(t, "val1", v, "expired and marked as such, but value is available")
	assert.False(t, ok)

	assert.Empty(t, lc.Values())
}

func TestCache_GetExpiration(t *testing.T) {
	lc := NewCache[string, string]().WithTTL(time.Second * 5)

	lc.Set("key1", "val1", time.Second*5)
	assert.Equal(t, 1, lc.Len())

	exp, ok := lc.GetExpiration("key1")
	assert.True(t, ok)
	assert.True(t, exp.After(time.Now().Add(time.Second*4)))
	assert.True(t, exp.Before(time.Now().Add(time.Second*6)))

	lc.Set("key2", "val2", time.Second*10)
	assert.Equal(t, 2, lc.Len())

	exp, ok = lc.GetExpiration("key2")
	assert.True(t, ok)
	assert.True(t, exp.After(time.Now().Add(time.Second*9)))
	assert.True(t, exp.Before(time.Now().Add(time.Second*11)))

	exp, ok = lc.GetExpiration("non-existing-key")
	assert.False(t, ok)
	assert.Zero(t, exp)
}
