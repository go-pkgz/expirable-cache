package cache_test

import (
	"fmt"
	"time"

	cache "github.com/go-pkgz/expirable-cache/v3"
)

func Example() {
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
