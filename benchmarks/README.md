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
BenchmarkGoCache_Set-8                              	13220007	        82.81 ns/op	      68 B/op	       1 allocs/op
BenchmarkGoCache_Get-8                              	17929201	        65.95 ns/op	       3 B/op	       0 allocs/op
BenchmarkGoCache_SetAndGet-8                        	16995872	        68.13 ns/op	      36 B/op	       1 allocs/op
BenchmarkTTLCache_Set-8                             	 3197972	       381.0 ns/op	       4 B/op	       0 allocs/op
BenchmarkTTLCache_Get-8                             	 6211912	       193.4 ns/op	      51 B/op	       1 allocs/op
BenchmarkTTLCache_SetAndGet-8                       	 5323972	       229.9 ns/op	      28 B/op	       1 allocs/op
BenchmarkExpirableCache_Set-8                       	18677211	        64.96 ns/op	       4 B/op	       0 allocs/op
BenchmarkExpirableCache_Get-8                       	15005173	        77.67 ns/op	       3 B/op	       0 allocs/op
BenchmarkExpirableCache_SetAndGet-8                 	18538854	        64.68 ns/op	       4 B/op	       0 allocs/op
BenchmarkRistretto_Set-8                            	 1531449	       777.2 ns/op	     262 B/op	       5 allocs/op
BenchmarkRistretto_Get-8                            	14814646	        81.51 ns/op	      27 B/op	       2 allocs/op
BenchmarkRistretto_SetAndGet-8                      	 2273522	       522.4 ns/op	     144 B/op	       3 allocs/op
BenchmarkGoCache_GetWithTypeAssertion-8             	18282196	        67.55 ns/op	       3 B/op	       0 allocs/op
BenchmarkTTLCache_GetWithoutTypeAssertion-8         	 6057862	       199.6 ns/op	      51 B/op	       1 allocs/op
BenchmarkExpirableCache_GetWithoutTypeAssertion-8   	15723604	        77.66 ns/op	       3 B/op	       0 allocs/op
BenchmarkRistretto_GetWithTypeAssertion-8           	14620899	        81.67 ns/op	      27 B/op	       2 allocs/op
BenchmarkGoCache_RealWorldScenario-8                	16225479	        70.83 ns/op	       4 B/op	       0 allocs/op
BenchmarkTTLCache_RealWorldScenario-8               	 6004263	       203.0 ns/op	      52 B/op	       1 allocs/op
BenchmarkExpirableCache_RealWorldScenario-8         	15008644	        80.00 ns/op	       3 B/op	       0 allocs/op
BenchmarkRistretto_RealWorldScenario-8              	11524798	        98.00 ns/op	      29 B/op	       2 allocs/op
```

## Methodology Note on Ristretto

Ristretto applies writes asynchronously: `Set` only queues the write and returns, and the value may
not be readable, or may be dropped altogether, until the queue is drained. Every benchmark here calls
`cache.Wait()` after a timed `Set`, so the reported write cost includes applying the write and the
numbers describe the same unit of work as the synchronous `Set` of the other libraries. Writes
rejected by the admission policy are counted and reported as a `drops/op` metric; there were none in
the run above.

## Summary of Results

| Operation | [go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache) | [patrickmn/go-cache](https://github.com/patrickmn/go-cache) | [jellydator/ttlcache](https://github.com/jellydator/ttlcache) | [dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) |
|-----------|-----------------|----------|----------|-----------|
| Set | 64.96 ns/op | 82.81 ns/op | 381.0 ns/op | 777.2 ns/op |
| Get | 77.67 ns/op | 65.95 ns/op | 193.4 ns/op | 81.51 ns/op |
| Set+Get | 64.68 ns/op | 68.13 ns/op | 229.9 ns/op | 522.4 ns/op |
| Real-world scenario | 80.00 ns/op | 70.83 ns/op | 203.0 ns/op | 98.00 ns/op |
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
   - Excellent for read-heavy workloads, Get is on par with the fastest libraries here
   - Much higher memory usage than other libraries
   - Considerably slower Set operations, and mixed read/write workloads suffer as well once queued writes are applied
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