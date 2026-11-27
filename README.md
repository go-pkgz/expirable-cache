# expirable-cache

[![Build Status](https://github.com/go-pkgz/expirable-cache/workflows/build-v3/badge.svg)](https://github.com/go-pkgz/expirable-cache/actions/workflows/ci-v3.yml)
[![Coverage Status](https://coveralls.io/repos/github/go-pkgz/expirable-cache/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/expirable-cache?branch=master)
[![godoc](https://godoc.org/github.com/go-pkgz/expirable-cache/v3?status.svg)](https://pkg.go.dev/github.com/go-pkgz/expirable-cache/v3?tab=doc)

Package cache implements expirable cache.

- Support LRC, LRU and TTL-based eviction.
- Package is thread-safe and doesn't spawn any goroutines. The eviction callback set with `WithOnEvicted` and the predicate passed to `InvalidateFn` run with the cache lock held and must not call the cache, or they deadlock.
- v3 `ContainsOrAdd` checks for a key and adds it if missing under one lock, so concurrent callers need no lock of their own around `Contains` and `Add`.
- When a new key is added (`Set`, `Add`, `ContainsOrAdd`), the cache removes the oldest entry if it has expired, unless both the cache and the new entry use the default TTL; updating an existing key does not.
- In case `WithMaxKeys` is set, cache deletes the oldest entry disregarding its expiration date to maintain the size,
either using LRC or LRU eviction.
- In case of default TTL (10 years) and default `WithMaxKeys` (0, unlimited) the cache will be truly unlimited
 and will never delete entries from itself automatically, unless entries are given their own TTL.

Latest version is v3, it requires Go 1.23 or later; v2 and v1 require Go 1.20 or later. Only v3 is actively developed.

**Important**: only reliable way of not having expired entries stuck in a cache is to
run cache.DeleteExpired periodically using [time.Ticker](https://golang.org/pkg/time/#Ticker),
advisable period is 1/2 of TTL.

This cache is heavily inspired by [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru) _simplelru_ implementation. v3 implements `simplelru.LRUCache` interface, so if you use only the functions of that interface, you can switch from `github.com/hashicorp/golang-lru/v2/simplelru` or `github.com/hashicorp/golang-lru/v2/expirable` without any changes in your code except for cache creation, which must include `WithLRU()` to keep the LRU eviction order. The behavior still differs in these points:

- A TTL of 0 or less makes entries expire at once, while `expirable.NewLRU` treats 0 as no expiry; here the default TTL (10 years) is the way to disable expiry
- On an expired key, `Get` returns the stored value with `false` and `Contains` returns `false`, while `expirable` returns the zero value from `Get` and `true` from `Contains` until it removes the entry
- Expired entries are not removed in the background, see the note on `DeleteExpired` above
- `Add` also returns `true` when it removed an expired entry, not only when the size limit forced an eviction
- `Resize(0)` removes the size limit, as in `expirable`, while in `simplelru` it removes every entry

Key differences are:

- Support LRC (Least Recently Created) in addition to LRU and TTL-based eviction. In LRC mode the eviction order is
defined by the time of the last write: setting an existing key moves it to the front, while reads leave the order
untouched, which is what LRU mode adds
- Supports per-key TTL setting
- Doesn't spawn any goroutines, whereas `hashicorp/golang-lru/v2/expirable` spawns goroutine which is never killed ([as of now](https://github.com/hashicorp/golang-lru/issues/159))
- Provides stats about hits and misses, added and evicted entries: `Added` counts new keys, not updates, and `Evicted` counts every entry removed for any reason, including explicit removal and `Purge`

### API

v3 is configured by chaining these methods on `NewCache[K, V]()`:

- `WithTTL(ttl)` sets the cache-wide TTL, used by `Add`, `ContainsOrAdd` and `Set` with a ttl of 0. The default is 10 years, which is the way to disable expiry; a TTL of 0 or less makes those entries expire at once
- `WithMaxKeys(n)` sets how many keys to keep. The default is 0, unlimited
- `WithLRU()` switches to LRU eviction, in which `Get` moves an entry to the front. The default is LRC, in which only adding or updating a key does
- `WithOnEvicted(fn)` sets a function called for every removed entry, whether removed automatically or manually. There is none by default

Entries are added with `Set` (own TTL), `Add` and `ContainsOrAdd` (cache-wide TTL); read with `Get`, `Peek`, `Contains`, `GetExpiration`, `GetOldest`, `Keys`, `Values` and `Len`; and removed with `Remove`, `Invalidate`, `InvalidateFn`, `RemoveOldest`, `DeleteExpired`, `Purge` and `Resize`. `Stat` returns the counters. Each method is documented on [pkg.go.dev](https://pkg.go.dev/github.com/go-pkgz/expirable-cache/v3#Cache).

### Older versions

v1 (`github.com/go-pkgz/expirable-cache`) stores `any` values under string keys and takes its options in the constructor: `cache.NewCache(cache.TTL(time.Minute), cache.MaxKeys(100), cache.LRU(), cache.OnEvicted(fn))` returns `(Cache, error)`. v2 (`github.com/go-pkgz/expirable-cache/v2`) has the chained `With*` methods of v3. Neither has `Add`, `Contains`, `ContainsOrAdd`, `Values`, `Remove`, `Resize`, `GetExpiration` or `GetOldest`, and their `RemoveOldest` returns nothing. On an expired entry, `Get` returns `nil` in v1 and the zero value in v2, with `false`, where v3 returns the stored value.

### Usage example

```go
package main

import (
	"fmt"
	"time"

	cache "github.com/go-pkgz/expirable-cache/v3"
)

func main() {
	// make cache with short TTL and 3 max keys
	c := cache.NewCache[string, string]().WithMaxKeys(3).WithTTL(time.Millisecond * 100)

	// set value under key1.
	// with 0 ttl (last parameter) will use cache-wide setting instead (100ms).
	c.Set("key1", "val1", 0)

	// get value under key1
	r, ok := c.Get("key1")
	fmt.Printf("value before expiration is found: %v, value: %v\n", ok, r)

	time.Sleep(time.Millisecond * 110)

	// get value under key1 after key expiration.
	// expired entry is not removed until something touches it, so the stored value
	// is still returned, with ok set to false
	r, ok = c.Get("key1")
	fmt.Printf("value after expiration is found: %v, value: %v\n", ok, r)

	// set value under key2, would evict key1 because it is already expired.
	// ttl (last parameter) overrides cache-wide ttl.
	c.Set("key2", "val2", time.Minute*5)

	fmt.Printf("%+v\n", c)

	// Output:
	// value before expiration is found: true, value: val1
	// value after expiration is found: false, value: val1
	// Size: 1, Stats: {Hits:1 Misses:1 Added:2 Evicted:1} (50.0%)
}
```

### Performance Comparison

For detailed benchmarks comparing different versions and cache implementations, see the [benchmarks](./benchmarks) directory.

Based on all the benchmarks across four different caching libraries:

1. **[go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache)** remains the best overall option:
   - Excellent performance across all operations
   - Lowest or tied for lowest allocations per operation
   - Type safety through generics
   - Clean API with method chaining
   - Simple implementation

2. **[dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)** is a strong contender for specific use cases:
   - Great performance for read-heavy workloads
   - Sophisticated memory management for very large caches
   - Built-in metrics and statistics
   - Designed for high-concurrency environments
   - Writes are asynchronous, so they are cheap to issue but expensive to complete, see the
     [methodology note](./benchmarks/README.md#methodology-note-on-ristretto) in the benchmarks

3. **[patrickmn/go-cache](https://github.com/patrickmn/go-cache)** is still the fastest for reads but lacks modern features

4. **[jellydator/ttlcache](https://github.com/jellydator/ttlcache)** lags behind in speed, two to four times slower than the leaders, although its allocations per operation are now on par with them.

#### Version Improvements

v2 and v3 use Go generics and achieve significant performance improvements over v1:

- v2 takes **29-32% less time** per operation than v1 in the table below, and makes one allocation per benchmark operation where v1 makes four
- v3 maintains the performance gains of v2 while being compatible with the Hashicorp `simplelru` interface
- The released v1.0.0 and v2.0.0 date from 2022 and predate these optimizations

#### Performance Comparison

The table was added in April 2025; the machine and Go version it was measured on were not recorded.

| Operation               | v1        | v2        | v3        | Improvement v1→v3 |
|-------------------------|-----------|-----------|-----------|-------------------|
| Random LRU (no expire)  | 188.3 ns/op | 127.5 ns/op | 132.3 ns/op | ~30% faster |
| Frequency LRU (no expire) | 180.3 ns/op | 127.4 ns/op | 128.1 ns/op | ~29% faster |
| Random LRU (with expire) | 191.9 ns/op | 129.7 ns/op | 130.7 ns/op | ~32% faster |
| Frequency LRU (with expire) | 181.3 ns/op | 126.7 ns/op | 131.2 ns/op | ~28% faster |

#### Cross-Library Comparison

Recent benchmarks comparing expirable-cache with other popular Go caching libraries:

| Operation | [go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache) | [patrickmn/go-cache](https://github.com/patrickmn/go-cache) | [jellydator/ttlcache](https://github.com/jellydator/ttlcache) | [dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) |
|-----------|-----------------|----------|----------|-----------|
| Set | 65.59 ns/op | 79.87 ns/op | 256.7 ns/op | 778.9 ns/op |
| Get | 80.98 ns/op | 67.82 ns/op | 191.0 ns/op | 78.75 ns/op |
| Set+Get | 65.93 ns/op | 69.63 ns/op | 198.9 ns/op | 528.7 ns/op |
| Real-world scenario | 82.28 ns/op | 70.87 ns/op | 189.5 ns/op | 105.6 ns/op |
| Memory allocations (Set) | 4 B/op | 67 B/op | 4 B/op | 262 B/op |
| Memory allocations (Get) | 3 B/op | 3 B/op | 3 B/op | 27 B/op |

For detailed benchmarks and methodology, see the [benchmarks directory](./benchmarks).