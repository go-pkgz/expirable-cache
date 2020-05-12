package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCacheNoPurge(t *testing.T) {
	lc, err := NewCache()
	assert.NoError(t, err)

	lc.Set("key1", "val1", 0)
	assert.Equal(t, 1, lc.Len())

	v, ok := lc.Peek("key1")
	assert.Equal(t, "val1", v)
	assert.True(t, ok)

	v, ok = lc.Peek("key2")
	assert.Empty(t, v)
	assert.False(t, ok)

	assert.Equal(t, []string{"key1"}, lc.Keys())
}

func TestCacheWithDeleteExpired(t *testing.T) {
	var evicted []string
	lc, err := NewCache(
		TTL(150*time.Millisecond),
		OnEvicted(func(key string, value interface{}) { evicted = append(evicted, key, value.(string)) }),
	)
	assert.NoError(t, err)

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
	assert.Nil(t, v)

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

func TestCacheWithPurgeEnforcedBySize(t *testing.T) {
	lc, err := NewCache(MaxKeys(10), TTL(time.Hour))
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		i := i
		lc.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i), 0)
		v, ok := lc.Get(fmt.Sprintf("key%d", i))
		assert.Equal(t, fmt.Sprintf("val%d", i), v)
		assert.True(t, ok)
		assert.True(t, lc.Len() < 20)
	}

	assert.Equal(t, 10, lc.Len())
}

func TestCacheConcurrency(t *testing.T) {
	lc, err := NewCache()
	assert.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(i int) {
			lc.Set(fmt.Sprintf("key-%d", i/10), fmt.Sprintf("val-%d", i/10), 0)
			wg.Done()
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 100, lc.Len())
}

func TestCacheInvalidateAndEvict(t *testing.T) {
	var evicted int
	lc, err := NewCache(LRU(), OnEvicted(func(_ string, _ interface{}) { evicted++ }))
	assert.NoError(t, err)

	lc.Set("key1", "val1", 0)
	lc.Set("key2", "val2", 0)

	val, ok := lc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "val1", val)
	assert.Equal(t, 0, evicted)

	lc.Invalidate("key1")
	assert.Equal(t, 1, evicted)
	val, ok = lc.Get("key1")
	assert.Empty(t, val)
	assert.False(t, ok)

	val, ok = lc.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, "val2", val)

	lc.InvalidateFn(func(key string) bool {
		return key == "key2"
	})
	assert.Equal(t, 2, evicted)
	_, ok = lc.Get("key2")
	assert.False(t, ok)
	assert.Equal(t, 0, lc.Len())
}

func TestCacheBadOption(t *testing.T) {
	lc, err := NewCache(func(lc *cacheImpl) error {
		return errors.New("mock err")
	})
	assert.EqualError(t, err, "failed to set cache option: mock err")
	assert.Nil(t, lc)
}

func TestCacheExpired(t *testing.T) {
	lc, err := NewCache(TTL(time.Millisecond * 5))
	assert.NoError(t, err)

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
	assert.Empty(t, v)
	assert.False(t, ok)

	v, ok = lc.Get("key1")
	assert.Empty(t, v)
	assert.False(t, ok)
}

func TestCacheRemoveOldest(t *testing.T) {
	lc, err := NewCache(LRU(), MaxKeys(2))
	assert.NoError(t, err)

	lc.Set("key1", "val1", 0)
	assert.Equal(t, 1, lc.Len())

	v, ok := lc.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "val1", v)

	assert.Equal(t, []string{"key1"}, lc.Keys())
	assert.Equal(t, 1, lc.Len())

	lc.Set("key2", "val2", 0)
	assert.Equal(t, []string{"key1", "key2"}, lc.Keys())
	assert.Equal(t, 2, lc.Len())

	lc.RemoveOldest()

	assert.Equal(t, []string{"key2"}, lc.Keys())
	assert.Equal(t, 1, lc.Len())
}

func ExampleCache() {
	// make cache with short TTL and 3 max keys
	cache, _ := NewCache(MaxKeys(3), TTL(time.Millisecond*10))

	// set value under key1.
	// with 0 ttl (last parameter) will use cache-wide setting instead (10ms).
	cache.Set("key1", "val1", 0)

	// get value under key1
	r, ok := cache.Get("key1")

	// check for OK value, because otherwise return would be nil and
	// type conversion will panic
	if ok {
		rstr := r.(string) // convert cached value from interface{} to real type
		fmt.Printf("value before expiration is found: %v, value: %v\n", ok, rstr)
	}

	time.Sleep(time.Millisecond * 11)

	// get value under key1 after key expiration
	r, ok = cache.Get("key1")
	// don't convert to string as with ok == false value would be nil
	fmt.Printf("value after expiration is found: %v, value: %v\n", ok, r)

	// set value under key2, would evict old entry because it is already expired.
	// ttl (last parameter) overrides cache-wide ttl.
	cache.Set("key2", "val2", time.Minute*5)

	fmt.Printf("%+v\n", cache)
	// Output:
	// value before expiration is found: true, value: val1
	// value after expiration is found: false, value: <nil>
	// Size: 1, Stats: {Hits:1 Misses:1 Added:2 Evicted:1} (50.0%)
}
