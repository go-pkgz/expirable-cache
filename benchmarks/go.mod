module github.com/go-pkgz/expirable-cache/benchmarks

go 1.23.0

require (
	github.com/dgraph-io/ristretto v0.2.0
	github.com/go-pkgz/expirable-cache/v3 v3.0.0
	github.com/jellydator/ttlcache/v3 v3.3.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
)

replace github.com/go-pkgz/expirable-cache/v3 => ../v3
