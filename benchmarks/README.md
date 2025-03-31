# expirable-cache benchmarks

This directory contains comprehensive benchmarks comparing performance across different caching libraries for Go.

## Libraries Compared

1. **[go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache)** (v3) - This library, uses generics and LRU/LRC eviction
2. **[patrickmn/go-cache](https://github.com/patrickmn/go-cache)** - Lightweight in-memory key:value store/cache with expiration support
3. **[jellydator/ttlcache](https://github.com/jellydator/ttlcache)** - An in-memory cache with expiration
4. **[dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)** - A high performance memory-bound Go cache from Dgraph

## Benchmark Results

Here are the results from running the benchmarks on an Apple M3 processor:

```
$ go test -bench=. -benchmem
goos: darwin
goarch: arm64
pkg: github.com/go-pkgz/expirable-cache/benchmarks
cpu: Apple M3
BenchmarkGoCache_Set-8                              12169099        82.67 ns/op       68 B/op       1 allocs/op
BenchmarkGoCache_Get-8                              18444500        63.81 ns/op        3 B/op       0 allocs/op
BenchmarkGoCache_SetAndGet-8                        17692840        67.94 ns/op       36 B/op       1 allocs/op
BenchmarkTTLCache_Set-8                              2707100       448.8 ns/op         5 B/op       0 allocs/op
BenchmarkTTLCache_Get-8                              6269760       190.9 ns/op        51 B/op       1 allocs/op
BenchmarkTTLCache_SetAndGet-8                        4806730       253.9 ns/op        28 B/op       1 allocs/op
BenchmarkExpirableCache_Set-8                       17192541        69.14 ns/op        4 B/op       0 allocs/op
BenchmarkExpirableCache_Get-8                       15239731        78.12 ns/op        3 B/op       0 allocs/op
BenchmarkExpirableCache_SetAndGet-8                 17954317        66.62 ns/op        4 B/op       0 allocs/op
BenchmarkRistretto_Set-8                             1456555       820.0 ns/op       262 B/op       5 allocs/op
BenchmarkRistretto_Get-8                            14321246        84.23 ns/op       27 B/op       2 allocs/op
BenchmarkRistretto_SetAndGet-8                       5989928       198.2 ns/op        96 B/op       2 allocs/op
BenchmarkGoCache_GetWithTypeAssertion-8             17971783        66.44 ns/op        3 B/op       0 allocs/op
BenchmarkTTLCache_GetWithoutTypeAssertion-8          6114782       197.9 ns/op        51 B/op       1 allocs/op
BenchmarkExpirableCache_GetWithoutTypeAssertion-8   15095240        78.39 ns/op        3 B/op       0 allocs/op
BenchmarkRistretto_GetWithTypeAssertion-8           14458339        83.09 ns/op       27 B/op       2 allocs/op
BenchmarkGoCache_RealWorldScenario-8                16622580        70.24 ns/op        4 B/op       0 allocs/op
BenchmarkTTLCache_RealWorldScenario-8                6038209       198.0 ns/op        52 B/op       1 allocs/op
BenchmarkExpirableCache_RealWorldScenario-8         14718102        78.83 ns/op        3 B/op       0 allocs/op
BenchmarkRistretto_RealWorldScenario-8              13030276        83.40 ns/op       28 B/op       2 allocs/op
```

## Summary of Results

| Operation | [go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache) | [patrickmn/go-cache](https://github.com/patrickmn/go-cache) | [jellydator/ttlcache](https://github.com/jellydator/ttlcache) | [dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) |
|-----------|-----------------|----------|----------|-----------|
| Set | 69.14 ns/op | 82.67 ns/op | 448.8 ns/op | 820.0 ns/op |
| Get | 78.12 ns/op | 63.81 ns/op | 190.9 ns/op | 84.23 ns/op |
| Set+Get | 66.62 ns/op | 67.94 ns/op | 253.9 ns/op | 198.2 ns/op |
| Real-world scenario | 78.83 ns/op | 70.24 ns/op | 198.0 ns/op | 83.40 ns/op |
| Memory allocations (Set) | 4 B/op | 68 B/op | 5 B/op | 262 B/op |
| Memory allocations (Get) | 3 B/op | 3 B/op | 51 B/op | 27 B/op |

## Analysis

1. **[go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache)**:
   - Best overall balance of performance and features
   - Fastest Set operations among all libraries
   - Very competitive Get operations
   - Lowest memory usage across all benchmarks
   - Type safety through generics
   - Clean API with method chaining

2. **[patrickmn/go-cache](https://github.com/patrickmn/go-cache)**:
   - Fastest Get operations
   - Very competitive overall performance
   - However, it's known to leak goroutines and lacks modern features
   - Higher memory usage for Set operations than expirable-cache

3. **[dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)**:
   - Excellent for read-heavy workloads
   - Much higher memory usage than other libraries
   - Considerably slower Set operations
   - Best suited for very large caches where sophisticated memory management is beneficial

4. **[jellydator/ttlcache](https://github.com/jellydator/ttlcache)**:
   - Significantly slower than other libraries for all operations
   - Higher memory usage for Get operations
   - Not recommended for performance-critical applications

Thanks to [@analytically](https://github.com/analytically) for the benchmark code and initial analysis!

## Running the Benchmarks

To run the benchmarks yourself:

```bash
go test -bench=. -benchmem
```

For more focused testing:

```bash
# Test only Set operations
go test -bench=Set -benchmem

# Test only expirable-cache
go test -bench=ExpirableCache -benchmem

# Test only real-world scenarios
go test -bench=RealWorldScenario -benchmem
```