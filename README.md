# expirable-cache

[![Build Status](https://github.com/go-pkgz/expirable-cache/workflows/build/badge.svg)](https://github.com/go-pkgz/expirable-cache/actions)
[![Coverage Status](https://coveralls.io/repos/github/go-pkgz/expirable-cache/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/expirable-cache?branch=master)
[![godoc](https://godoc.org/github.com/go-pkgz/expirable-cache?status.svg)](https://pkg.go.dev/github.com/go-pkgz/expirable-cache?tab=doc)

Package cache implements expirable cache.

- Support LRC, LRU and TTL-based eviction.
- Package is thread-safe and doesn't spawn any goroutines.
- On every Set() call, cache deletes single oldest entry in case it's expired.
- In case MaxSize is set, cache deletes the oldest entry disregarding its expiration date to maintain the size,
either using LRC or LRU eviction.
- In case of default TTL (10 years) and default MaxSize (0, unlimited) the cache will be truly unlimited
 and will never delete entries from itself automatically.

**Important**: only reliable way of not having expired entries stuck in a cache is to
run cache.DeleteExpired periodically using [time.Ticker](https://golang.org/pkg/time/#Ticker),
advisable period is 1/2 of TTL.

This cache is heavily inspired by [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru) _simplelru_ implementation. v3 implements `simplelru.LRUCache` interface, so if you use a subset of functions, so you can switch from `github.com/hashicorp/golang-lru/v2/simplelru` or `github.com/hashicorp/golang-lru/v2/expirable` without any changes in your code except for cache creation. Key differences are:

- Support LRC (Least Recently Created) in addition to LRU and TTL-based eviction
- Supports per-key TTL setting
- Doesn't spawn any goroutines, whereas `hashicorp/golang-lru/v2/expirable` spawns goroutine which is never killed ([as of now](https://github.com/hashicorp/golang-lru/issues/159))
- Provides stats about hits and misses, added and evicted entries

### Usage example

```go
package main

import (
	"fmt"
	"time"

	"github.com/go-pkgz/expirable-cache/v3"
)

func main() {
	// make cache with short TTL and 3 max keys
	c := cache.NewCache[string, string]().WithMaxKeys(3).WithTTL(time.Millisecond * 10)

	// set value under key1.
	// with 0 ttl (last parameter) will use cache-wide setting instead (10ms).
	c.Set("key1", "val1", 0)

	// get value under key1
	r, ok := c.Get("key1")

	// check for OK value, because otherwise return would be nil and
	// type conversion will panic
	if ok {
		rstr := r.(string) // convert cached value from interface{} to real type
		fmt.Printf("value before expiration is found: %v, value: %v\n", ok, rstr)
	}

	time.Sleep(time.Millisecond * 11)

	// get value under key1 after key expiration
	r, ok = c.Get("key1")
	// don't convert to string as with ok == false value would be nil
	fmt.Printf("value after expiration is found: %v, value: %v\n", ok, r)

	// set value under key2, would evict old entry because it is already expired.
	// ttl (last parameter) overrides cache-wide ttl.
	c.Set("key2", "val2", time.Minute*5)

	fmt.Printf("%+v\n", c)
	// Output:
	// value before expiration is found: true, value: val1
	// value after expiration is found: false, value: <nil>
	// Size: 1, Stats: {Hits:1 Misses:1 Added:2 Evicted:1} (50.0%)
}
```

### Performance Comparison

For detailed benchmarks comparing different versions and cache implementations, see the [benchmarks](./benchmarks) directory.

Based on all the benchmarks across four different caching libraries:

1. **[go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache)** remains the best overall option:
   - Excellent performance across all operations
   - Lowest memory usage and allocations
   - Type safety through generics
   - Clean API with method chaining
   - Simple implementation

2. **[dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)** is a strong contender for specific use cases:
   - Great performance for read-heavy workloads
   - Sophisticated memory management for very large caches
   - Built-in metrics and statistics
   - Designed for high-concurrency environments

3. **[patrickmn/go-cache](https://github.com/patrickmn/go-cache)** is still fastest for pure raw performance but lacks modern features, and leaks goroutines

4. **[jellydator/ttlcache](https://github.com/jellydator/ttlcache)** lags behind in performance compared to all other options.

#### Version Improvements

v2 and v3 use Go generics and achieve significant performance improvements over v1:

- v2 is approximately **28-42% faster** than v1 for basic operations
- v3 maintains the performance gains of v2 while being compatible with the Hashicorp `simplelru` interface
- Recent optimizations have improved performance across all versions

#### Performance Comparison

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
| Set | 69.14 ns/op | 82.67 ns/op | 448.8 ns/op | 820.0 ns/op |
| Get | 78.12 ns/op | 63.81 ns/op | 190.9 ns/op | 84.23 ns/op |
| Set+Get | 66.62 ns/op | 67.94 ns/op | 253.9 ns/op | 198.2 ns/op |
| Real-world scenario | 78.83 ns/op | 70.24 ns/op | 198.0 ns/op | 83.40 ns/op |
| Memory allocations | Lowest | Low | Medium | Highest |

<details> 
<summary>v1 benchmark results</summary>

```
~/expirable-cache ❯ go test -bench=.
goos: darwin
goarch: arm64
pkg: github.com/go-pkgz/expirable-cache
BenchmarkLRU_Rand_NoExpire-8     	 4494738	       272.4 ns/op
--- BENCH: BenchmarkLRU_Rand_NoExpire-8
    cache_test.go:46: hit: 0 miss: 1 ratio: 0.000000
    cache_test.go:46: hit: 1 miss: 99 ratio: 0.010000
    cache_test.go:46: hit: 1352 miss: 8648 ratio: 0.135200
    cache_test.go:46: hit: 248678 miss: 751322 ratio: 0.248678
    cache_test.go:46: hit: 1121791 miss: 3372947 ratio: 0.249579
BenchmarkLRU_Freq_NoExpire-8     	 4612648	       261.6 ns/op
--- BENCH: BenchmarkLRU_Freq_NoExpire-8
    cache_test.go:74: hit: 1 miss: 0 ratio: 1.000000
    cache_test.go:74: hit: 100 miss: 0 ratio: 1.000000
    cache_test.go:74: hit: 9825 miss: 175 ratio: 0.982500
    cache_test.go:74: hit: 312345 miss: 687655 ratio: 0.312345
    cache_test.go:74: hit: 1414620 miss: 3198028 ratio: 0.306683
BenchmarkLRU_Rand_WithExpire-8   	 4109704	       286.5 ns/op
--- BENCH: BenchmarkLRU_Rand_WithExpire-8
    cache_test.go:99: hit: 0 miss: 1 ratio: 0.000000
    cache_test.go:99: hit: 0 miss: 100 ratio: 0.000000
    cache_test.go:99: hit: 1304 miss: 8696 ratio: 0.130400
    cache_test.go:99: hit: 248310 miss: 751690 ratio: 0.248310
    cache_test.go:99: hit: 1027317 miss: 3082387 ratio: 0.249973
BenchmarkLRU_Freq_WithExpire-8   	 4341217	       279.6 ns/op
--- BENCH: BenchmarkLRU_Freq_WithExpire-8
    cache_test.go:127: hit: 1 miss: 0 ratio: 1.000000
    cache_test.go:127: hit: 100 miss: 0 ratio: 1.000000
    cache_test.go:127: hit: 9868 miss: 132 ratio: 0.986800
    cache_test.go:127: hit: 38221 miss: 961779 ratio: 0.038221
    cache_test.go:127: hit: 37296 miss: 4303921 ratio: 0.008591
PASS
ok  	github.com/go-pkgz/expirable-cache	18.307s
```
</details>

<details> 
<summary>v3 benchmark results</summary>

```
~/Desktop/expirable-cache/v3 master !2 ❯ go test -bench=.
goos: darwin
goarch: arm64
pkg: github.com/go-pkgz/expirable-cache/v3
BenchmarkLRU_Rand_NoExpire-8     	 7556680	       158.1 ns/op
--- BENCH: BenchmarkLRU_Rand_NoExpire-8
    cache_test.go:47: hit: 0 miss: 1 ratio: 0.000000
    cache_test.go:47: hit: 0 miss: 100 ratio: 0.000000
    cache_test.go:47: hit: 1409 miss: 8591 ratio: 0.140900
    cache_test.go:47: hit: 249063 miss: 750937 ratio: 0.249063
    cache_test.go:47: hit: 1887563 miss: 5669117 ratio: 0.249787
BenchmarkLRU_Freq_NoExpire-8     	 7876738	       150.9 ns/op
--- BENCH: BenchmarkLRU_Freq_NoExpire-8
    cache_test.go:75: hit: 1 miss: 0 ratio: 1.000000
    cache_test.go:75: hit: 100 miss: 0 ratio: 1.000000
    cache_test.go:75: hit: 9850 miss: 150 ratio: 0.985000
    cache_test.go:75: hit: 310888 miss: 689112 ratio: 0.310888
    cache_test.go:75: hit: 2413312 miss: 5463426 ratio: 0.306385
BenchmarkLRU_Rand_WithExpire-8   	 6822362	       175.3 ns/op
--- BENCH: BenchmarkLRU_Rand_WithExpire-8
    cache_test.go:100: hit: 0 miss: 1 ratio: 0.000000
    cache_test.go:100: hit: 0 miss: 100 ratio: 0.000000
    cache_test.go:100: hit: 1326 miss: 8674 ratio: 0.132600
    cache_test.go:100: hit: 248508 miss: 751492 ratio: 0.248508
    cache_test.go:100: hit: 1704172 miss: 5118190 ratio: 0.249792
BenchmarkLRU_Freq_WithExpire-8   	 7098261	       168.1 ns/op
--- BENCH: BenchmarkLRU_Freq_WithExpire-8
    cache_test.go:128: hit: 1 miss: 0 ratio: 1.000000
    cache_test.go:128: hit: 100 miss: 0 ratio: 1.000000
    cache_test.go:128: hit: 9842 miss: 158 ratio: 0.984200
    cache_test.go:128: hit: 90167 miss: 909833 ratio: 0.090167
    cache_test.go:128: hit: 90421 miss: 7007840 ratio: 0.012738
PASS
ok  	github.com/go-pkgz/expirable-cache/v3	24.315s
```
</details>

<details> 

For detailed benchmarks and methodology, see the [benchmarks directory](./benchmarks).