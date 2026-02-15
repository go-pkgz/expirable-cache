package cache

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/simplelru"
	"github.com/stretchr/testify/assert"
)

func getRand(tb testing.TB) int64 {
	out, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		tb.Fatal(err)
	}
	return out.Int64()
}

func BenchmarkLRU_Rand_NoExpire(b *testing.B) {
	l := NewCache[int64, int64]().WithLRU().WithMaxKeys(8192)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkLRU_Freq_NoExpire(b *testing.B) {
	l := NewCache[int64, int64]().WithLRU().WithMaxKeys(8192)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkLRU_Rand_WithExpire(b *testing.B) {
	l := NewCache[int64, int64]().WithLRU().WithMaxKeys(8192).WithTTL(time.Millisecond * 10)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkLRU_Freq_WithExpire(b *testing.B) {
	l := NewCache[int64, int64]().WithLRU().WithMaxKeys(8192).WithTTL(time.Millisecond * 10)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func TestSimpleLRUInterface(_ *testing.T) {
	var _ simplelru.LRUCache[int, int] = NewCache[int, int]()
}

func TestCacheNoPurge(t *testing.T) {
	lc := NewCache[string, string]()

	k, v, ok := lc.GetOldest()
	assert.Empty(t, k)
	assert.Empty(t, v)
	assert.False(t, ok)

	lc.Add("key1", "val1")
	assert.Equal(t, 1, lc.Len())
	assert.True(t, lc.Contains("key1"))
	assert.False(t, lc.Contains("key2"))

	v, ok = lc.Peek("key1")
	assert.Equal(t, "val1", v)
	assert.True(t, ok)

	k, v, ok = lc.GetOldest()
	assert.Equal(t, "key1", k)
	assert.Equal(t, "val1", v)
	assert.True(t, ok)

	lc.Add("key3", "val3")
	lc.Add("key4", "val4")
	lc.Peek("key3")
	k, v, ok = lc.GetOldest()
	assert.Equal(t, "key1", k)
	assert.Equal(t, "val1", v)
	assert.True(t, ok)

	lc.Add("key1", "val1")
	k, v, ok = lc.GetOldest()
	assert.Equal(t, "key3", k)
	assert.Equal(t, "val3", v)
	assert.True(t, ok)

	v, ok = lc.Peek("key2")
	assert.Empty(t, v)
	assert.False(t, ok)

	assert.Equal(t, []string{"key3", "key4", "key1"}, lc.Keys())
}

func TestCache_Values(t *testing.T) {
	lc := NewCache[string, string]().WithMaxKeys(3)

	lc.Add("key1", "val1")
	lc.Add("key2", "val2")
	lc.Add("key3", "val3")

	values := lc.Values()
	if !reflect.DeepEqual(values, []string{"val1", "val2", "val3"}) {
		t.Fatalf("values differs from expected")
	}

	assert.Equal(t, 0, lc.Resize(0))
	assert.Equal(t, 1, lc.Resize(2))
	assert.Equal(t, 0, lc.Resize(5))
	assert.Equal(t, 1, lc.Resize(1))
}

func TestCacheWithPurgeEnforcedBySize(t *testing.T) {
	lc := NewCache[string, string]().WithTTL(time.Hour).WithMaxKeys(10)

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
	lc := NewCache[string, string]()
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
	lc := NewCache[string, string]().WithLRU().WithOnEvicted(func(_ string, _ string) { evicted++ })

	lc.Set("key1", "val1", 0)
	lc.Set("key2", "val2", 0)
	lc.Set("key3", "val3", 0)

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
	assert.Equal(t, 1, lc.Len())

	assert.True(t, lc.Remove("key3"))
	assert.Equal(t, 3, evicted)
	val, ok = lc.Get("key3")
	assert.Empty(t, val)
	assert.False(t, ok)
	assert.False(t, lc.Remove("key3"))
	assert.Zero(t, lc.Len())
}

func TestCacheRemoveOldest(t *testing.T) {
	lc := NewCache[string, string]().WithLRU().WithMaxKeys(2)

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

	k, v, ok := lc.RemoveOldest()
	assert.Equal(t, "key1", k)
	assert.Equal(t, "val1", v)
	assert.True(t, ok)

	assert.Equal(t, []string{"key2"}, lc.Keys())
	assert.Equal(t, 1, lc.Len())

	k, v, ok = lc.RemoveOldest()
	assert.Equal(t, "key2", k)
	assert.Equal(t, "val2", v)
	assert.True(t, ok)

	k, v, ok = lc.RemoveOldest()
	assert.Empty(t, k)
	assert.Empty(t, v)
	assert.False(t, ok)

	assert.Empty(t, lc.Keys())

}

func TestCacheContainsOrAdd(t *testing.T) {
	lc := NewCache[string, string]().WithLRU().WithMaxKeys(2)

	lc.Add("key1", "val1")
	assert.Equal(t, 1, lc.Len())

	lc.Add("key2", "val2")
	assert.Equal(t, 2, lc.Len())

	contains, evicted := lc.ContainsOrAdd("key1", "val1")
	assert.Equal(t, true, contains)
	assert.Equal(t, false, evicted)

	contains, evicted = lc.ContainsOrAdd("key3", "val3")
	assert.Equal(t, false, contains)
	assert.Equal(t, true, evicted)

	val, ok := lc.Get("key3")
	assert.Equal(t, true, ok)
	assert.Equal(t, "val3", val)

	// Make sure key1 evicted
	_, ok = lc.Get("key1")
	assert.Equal(t, false, ok)
}

func ExampleCache() {
	// make cache with short TTL and 3 max keys
	cache := NewCache[string, string]().WithMaxKeys(3).WithTTL(time.Millisecond * 10)

	// set value under key1.
	// with 0 ttl (last parameter) will use cache-wide setting instead (10ms).
	cache.Set("key1", "val1", 0)

	// get value under key1
	r, ok := cache.Get("key1")

	// check for OK value, because otherwise return would be nil and
	// type conversion will panic
	if ok {
		fmt.Printf("value before expiration is found: %v, value: %q\n", ok, r)
	}

	time.Sleep(time.Millisecond * 11)

	// get value under key1 after key expiration
	r, ok = cache.Get("key1")
	// don't convert to string as with ok == false value would be nil
	fmt.Printf("value after expiration is found: %v, value: %q\n", ok, r)

	// set value under key2, would evict old entry because it is already expired.
	// ttl (last parameter) overrides cache-wide ttl.
	cache.Set("key2", "val2", time.Minute*5)

	fmt.Printf("%+v\n", cache)
	// Output:
	// value before expiration is found: true, value: "val1"
	// value after expiration is found: false, value: "val1"
	// Size: 1, Stats: {Hits:1 Misses:1 Added:2 Evicted:1} (50.0%)
}
