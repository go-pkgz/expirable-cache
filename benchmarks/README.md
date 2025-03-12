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
BenchmarkGoCache_Set-8                             13150846        84.63 ns/op      68 B/op      1 allocs/op
BenchmarkGoCache_Get-8                             17746971        66.29 ns/op       3 B/op      0 allocs/op
BenchmarkGoCache_SetAndGet-8                       17334808        68.97 ns/op      36 B/op      1 allocs/op
BenchmarkTTLCache_Set-8                             2818526       430.7 ns/op        4 B/op      0 allocs/op
BenchmarkTTLCache_Get-8                             6197169       193.1 ns/op       51 B/op      1 allocs/op
BenchmarkTTLCache_SetAndGet-8                       4923871       242.8 ns/op       28 B/op      1 allocs/op
BenchmarkExpirableCache_Set-8                      17001220        69.65 ns/op       4 B/op      0 allocs/op
BenchmarkExpirableCache_Get-8                      15427915        78.27 ns/op       3 B/op      0 allocs/op
BenchmarkExpirableCache_SetAndGet-8                17600006        67.69 ns/op       4 B/op      0 allocs/op
BenchmarkRistretto_Set-8                            1502190       793.8 ns/op      262 B/op      5 allocs/op
BenchmarkRistretto_Get-8                           14158246        82.61 ns/op      27 B/op      2 allocs/op
BenchmarkRistretto_SetAndGet-8                      6852046       197.6 ns/op       96 B/op      2 allocs/op
BenchmarkGoCache_GetWithTypeAssertion-8            17942817        67.64 ns/op       3 B/op      0 allocs/op
BenchmarkTTLCache_GetWithoutTypeAssertion-8         6029751       199.0 ns/op       51 B/op      1 allocs/op
BenchmarkExpirableCache_GetWithoutTypeAssertion-8  15133410        79.33 ns/op       3 B/op      0 allocs/op
BenchmarkRistretto_GetWithTypeAssertion-8          14187883        86.12 ns/op      27 B/op      2 allocs/op
BenchmarkGoCache_RealWorldScenario-8               16314248        70.60 ns/op       4 B/op      0 allocs/op
BenchmarkTTLCache_RealWorldScenario-8               5891400       200.0 ns/op       52 B/op      1 allocs/op
BenchmarkExpirableCache_RealWorldScenario-8        14579878        79.98 ns/op       3 B/op      0 allocs/op
BenchmarkRistretto_RealWorldScenario-8             12897400        85.88 ns/op      28 B/op      2 allocs/op
```

## Summary of Results

| Operation | [go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache) | [patrickmn/go-cache](https://github.com/patrickmn/go-cache) | [jellydator/ttlcache](https://github.com/jellydator/ttlcache) | [dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) |
|-----------|-----------------|----------|----------|-----------|
| Set | 69.65 ns/op | 84.63 ns/op | 430.7 ns/op | 793.8 ns/op |
| Get | 78.27 ns/op | 66.29 ns/op | 193.1 ns/op | 82.61 ns/op |
| Set+Get | 67.69 ns/op | 68.97 ns/op | 242.8 ns/op | 197.6 ns/op |
| Real-world scenario | 79.98 ns/op | 70.60 ns/op | 200.0 ns/op | 85.88 ns/op |
| Memory allocations (Set) | 4 B/op | 68 B/op | 4 B/op | 262 B/op |
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