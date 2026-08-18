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
BenchmarkGoCache_Set-8                              	14435580	        79.87 ns/op	      67 B/op	       1 allocs/op
BenchmarkGoCache_Get-8                              	18228678	        67.82 ns/op	       3 B/op	       0 allocs/op
BenchmarkGoCache_SetAndGet-8                        	17382297	        69.63 ns/op	      36 B/op	       1 allocs/op
BenchmarkTTLCache_Set-8                             	 4442306	       256.7 ns/op	       4 B/op	       0 allocs/op
BenchmarkTTLCache_Get-8                             	 6576498	       191.0 ns/op	       3 B/op	       0 allocs/op
BenchmarkTTLCache_SetAndGet-8                       	 6569042	       198.9 ns/op	       4 B/op	       0 allocs/op
BenchmarkExpirableCache_Set-8                       	17978492	        65.59 ns/op	       4 B/op	       0 allocs/op
BenchmarkExpirableCache_Get-8                       	14897770	        80.98 ns/op	       3 B/op	       0 allocs/op
BenchmarkExpirableCache_SetAndGet-8                 	18371962	        65.93 ns/op	       4 B/op	       0 allocs/op
BenchmarkRistretto_Set-8                            	 1540219	       778.9 ns/op	     262 B/op	       5 allocs/op
BenchmarkRistretto_Get-8                            	14860327	        78.75 ns/op	      27 B/op	       2 allocs/op
BenchmarkRistretto_SetAndGet-8                      	 2286632	       528.7 ns/op	     144 B/op	       3 allocs/op
BenchmarkGoCache_GetWithTypeAssertion-8             	18340864	        67.55 ns/op	       3 B/op	       0 allocs/op
BenchmarkTTLCache_GetWithoutTypeAssertion-8         	 6430896	       190.9 ns/op	       3 B/op	       0 allocs/op
BenchmarkExpirableCache_GetWithoutTypeAssertion-8   	15096373	        77.74 ns/op	       3 B/op	       0 allocs/op
BenchmarkRistretto_GetWithTypeAssertion-8           	14743591	        80.04 ns/op	      27 B/op	       2 allocs/op
BenchmarkGoCache_RealWorldScenario-8                	16369579	        70.87 ns/op	       4 B/op	       0 allocs/op
BenchmarkTTLCache_RealWorldScenario-8               	 6387519	       189.5 ns/op	       4 B/op	       0 allocs/op
BenchmarkExpirableCache_RealWorldScenario-8         	15014247	        82.28 ns/op	       3 B/op	       0 allocs/op
BenchmarkRistretto_RealWorldScenario-8              	11046955	       105.6 ns/op	      29 B/op	       2 allocs/op
```

## Methodology Note on Ristretto

Ristretto applies writes asynchronously: `Set` only queues the write and returns, and the value may
not be readable, or may be dropped altogether, until the queue is drained. Every benchmark here calls
`cache.Wait()` after a timed `Set`, so the reported write cost includes applying the write and the
numbers describe the same unit of work as the synchronous `Set` of the other libraries. Writes that
`Set` refuses because the write buffer is full are counted and reported as a `drops/op` metric; there
were none in the run above. A write that `Set` accepts can still be discarded later by the admission
policy, so the metric is a lower bound.

## Summary of Results

| Operation | [go-pkgz/expirable-cache](https://github.com/go-pkgz/expirable-cache) | [patrickmn/go-cache](https://github.com/patrickmn/go-cache) | [jellydator/ttlcache](https://github.com/jellydator/ttlcache) | [dgraph-io/ristretto](https://github.com/dgraph-io/ristretto) |
|-----------|-----------------|----------|----------|-----------|
| Set | 65.59 ns/op | 79.87 ns/op | 256.7 ns/op | 778.9 ns/op |
| Get | 80.98 ns/op | 67.82 ns/op | 191.0 ns/op | 78.75 ns/op |
| Set+Get | 65.93 ns/op | 69.63 ns/op | 198.9 ns/op | 528.7 ns/op |
| Real-world scenario | 82.28 ns/op | 70.87 ns/op | 189.5 ns/op | 105.6 ns/op |
| Memory allocations (Set) | 4 B/op | 67 B/op | 4 B/op | 262 B/op |
| Memory allocations (Get) | 3 B/op | 3 B/op | 3 B/op | 27 B/op |

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
   - Two to three times slower than go-cache and expirable-cache for all operations
   - Memory usage is on par with the leaders since v3.4.1
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