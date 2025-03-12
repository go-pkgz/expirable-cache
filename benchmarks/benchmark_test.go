package test

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	expcache "github.com/go-pkgz/expirable-cache/v3"
	"github.com/jellydator/ttlcache/v3"
	gocache "github.com/patrickmn/go-cache"
)

const (
	numItems = 10000
)

type testItem struct {
	ID    int
	Name  string
	Value float64
	Data  []byte
}

func generateRandomItems(n int) []testItem {
	items := make([]testItem, n)
	for i := 0; i < n; i++ {
		items[i] = testItem{
			ID:    i,
			Name:  fmt.Sprintf("item-%d", i),
			Value: rand.Float64() * 100,
			Data:  make([]byte, 100),
		}
		rand.Read(items[i].Data)
	}
	return items
}

// Benchmarks for go-cache
func BenchmarkGoCache_Set(b *testing.B) {
	cache := gocache.New(5*time.Minute, 10*time.Minute)
	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := items[i%numItems]
		cache.Set(strconv.Itoa(item.ID), item, gocache.DefaultExpiration)
	}
}

func BenchmarkGoCache_Get(b *testing.B) {
	cache := gocache.New(5*time.Minute, 10*time.Minute)
	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, gocache.DefaultExpiration)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		_, _ = cache.Get(key)
	}
}

func BenchmarkGoCache_SetAndGet(b *testing.B) {
	cache := gocache.New(5*time.Minute, 10*time.Minute)
	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			item := items[i%numItems]
			cache.Set(strconv.Itoa(item.ID), item, gocache.DefaultExpiration)
		} else {
			key := strconv.Itoa(rand.Intn(numItems))
			_, _ = cache.Get(key)
		}
	}
}

// Benchmarks for ttlcache
func BenchmarkTTLCache_Set(b *testing.B) {
	cache := ttlcache.New[string, testItem](
		ttlcache.WithTTL[string, testItem](5 * time.Minute),
	)
	go cache.Start()
	defer cache.Stop()

	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := items[i%numItems]
		cache.Set(strconv.Itoa(item.ID), item, ttlcache.DefaultTTL)
	}
}

func BenchmarkTTLCache_Get(b *testing.B) {
	cache := ttlcache.New[string, testItem](
		ttlcache.WithTTL[string, testItem](5 * time.Minute),
	)
	go cache.Start()
	defer cache.Stop()

	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, ttlcache.DefaultTTL)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		_ = cache.Get(key)
	}
}

func BenchmarkTTLCache_SetAndGet(b *testing.B) {
	cache := ttlcache.New[string, testItem](
		ttlcache.WithTTL[string, testItem](5 * time.Minute),
	)
	go cache.Start()
	defer cache.Stop()

	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			item := items[i%numItems]
			cache.Set(strconv.Itoa(item.ID), item, ttlcache.DefaultTTL)
		} else {
			key := strconv.Itoa(rand.Intn(numItems))
			_ = cache.Get(key)
		}
	}
}

// Benchmarks for expirable-cache
func BenchmarkExpirableCache_Set(b *testing.B) {
	cache := expcache.NewCache[string, testItem]().WithTTL(5 * time.Minute)
	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := items[i%numItems]
		cache.Set(strconv.Itoa(item.ID), item, 0) // use default TTL
	}
}

func BenchmarkExpirableCache_Get(b *testing.B) {
	cache := expcache.NewCache[string, testItem]().WithTTL(5 * time.Minute)
	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		_, _ = cache.Get(key)
	}
}

func BenchmarkExpirableCache_SetAndGet(b *testing.B) {
	cache := expcache.NewCache[string, testItem]().WithTTL(5 * time.Minute)
	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			item := items[i%numItems]
			cache.Set(strconv.Itoa(item.ID), item, 0)
		} else {
			key := strconv.Itoa(rand.Intn(numItems))
			_, _ = cache.Get(key)
		}
	}
}

// Benchmarks for Ristretto cache
func BenchmarkRistretto_Set(b *testing.B) {
	// Create a new cache with a size of 100MB
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (100MB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	items := generateRandomItems(numItems)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := items[i%numItems]
		cache.Set(strconv.Itoa(item.ID), item, 1)
		cache.Wait() // ensure item is set before next operation
	}
}

func BenchmarkRistretto_Get(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (100MB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, 1)
	}
	cache.Wait() // ensure all items are set

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		_, _ = cache.Get(key)
	}
}

func BenchmarkRistretto_SetAndGet(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (100MB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	items := generateRandomItems(numItems)

	// Populate cache so gets have something to find
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, 1)
	}
	cache.Wait() // ensure all items are set

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			item := items[i%numItems]
			cache.Set(strconv.Itoa(item.ID), item, 1)
		} else {
			key := strconv.Itoa(rand.Intn(numItems))
			_, _ = cache.Get(key)
		}
	}
}

// Benchmark interface{} vs generic access patterns
func BenchmarkGoCache_GetWithTypeAssertion(b *testing.B) {
	cache := gocache.New(5*time.Minute, 10*time.Minute)
	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, gocache.DefaultExpiration)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if val, found := cache.Get(key); found {
			_ = val.(testItem)
		}
	}
}

func BenchmarkTTLCache_GetWithoutTypeAssertion(b *testing.B) {
	cache := ttlcache.New[string, testItem](
		ttlcache.WithTTL[string, testItem](5 * time.Minute),
	)
	go cache.Start()
	defer cache.Stop()

	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, ttlcache.DefaultTTL)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if item := cache.Get(key); item != nil {
			_ = item.Value()
		}
	}
}

func BenchmarkExpirableCache_GetWithoutTypeAssertion(b *testing.B) {
	cache := expcache.NewCache[string, testItem]().WithTTL(5 * time.Minute)
	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if val, found := cache.Get(key); found {
			_ = val
		}
	}
}

func BenchmarkRistretto_GetWithTypeAssertion(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (100MB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	items := generateRandomItems(numItems)

	// Populate cache
	for _, item := range items {
		cache.Set(strconv.Itoa(item.ID), item, 1)
	}
	cache.Wait() // ensure all items are set

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if val, found := cache.Get(key); found {
			_ = val.(testItem)
		}
	}
}

// Real-world scenario benchmark
func BenchmarkGoCache_RealWorldScenario(b *testing.B) {
	cache := gocache.New(5*time.Minute, 10*time.Minute)
	items := generateRandomItems(numItems)

	// Populate cache with 80% of items
	for i := 0; i < int(float64(numItems)*0.8); i++ {
		cache.Set(strconv.Itoa(items[i].ID), items[i], gocache.DefaultExpiration)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if val, found := cache.Get(key); found {
			_ = val.(testItem)
		} else {
			// Cache miss, add to cache
			index := rand.Intn(numItems)
			cache.Set(strconv.Itoa(index), items[index], gocache.DefaultExpiration)
		}
	}
}

func BenchmarkTTLCache_RealWorldScenario(b *testing.B) {
	cache := ttlcache.New[string, testItem](
		ttlcache.WithTTL[string, testItem](5 * time.Minute),
	)
	go cache.Start()
	defer cache.Stop()

	items := generateRandomItems(numItems)

	// Populate cache with 80% of items
	for i := 0; i < int(float64(numItems)*0.8); i++ {
		cache.Set(strconv.Itoa(items[i].ID), items[i], ttlcache.DefaultTTL)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if item := cache.Get(key); item != nil {
			_ = item.Value()
		} else {
			// Cache miss, add to cache
			index := rand.Intn(numItems)
			cache.Set(strconv.Itoa(index), items[index], ttlcache.DefaultTTL)
		}
	}
}

func BenchmarkExpirableCache_RealWorldScenario(b *testing.B) {
	cache := expcache.NewCache[string, testItem]().WithTTL(5 * time.Minute)
	items := generateRandomItems(numItems)

	// Populate cache with 80% of items
	for i := 0; i < int(float64(numItems)*0.8); i++ {
		cache.Set(strconv.Itoa(items[i].ID), items[i], 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if val, found := cache.Get(key); found {
			_ = val
		} else {
			// Cache miss, add to cache
			index := rand.Intn(numItems)
			cache.Set(strconv.Itoa(index), items[index], 0)
		}
	}
}

func BenchmarkRistretto_RealWorldScenario(b *testing.B) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (100MB)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		b.Fatal(err)
	}
	defer cache.Close()
	items := generateRandomItems(numItems)

	// Populate cache with 80% of items
	for i := 0; i < int(float64(numItems)*0.8); i++ {
		cache.Set(strconv.Itoa(items[i].ID), items[i], 1)
	}
	cache.Wait() // ensure all items are set

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(rand.Intn(numItems))
		if val, found := cache.Get(key); found {
			_ = val.(testItem)
		} else {
			// Cache miss, add to cache
			index := rand.Intn(numItems)
			cache.Set(strconv.Itoa(index), items[index], 1)
		}
	}
}
