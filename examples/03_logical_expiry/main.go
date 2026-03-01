package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot"
	"github.com/yikakia/cachalot/core/cache"
	store_ristretto "github.com/yikakia/cachalot/stores/ristretto"
)

func main() {
	ctx := context.Background()
	minute := time.Second // simulate minutes with seconds for a fast demo

	client, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	store := store_ristretto.New(client, store_ristretto.WithStoreName("logic-expire-ristretto"))

	var loaderCalls atomic.Int32
	loadFn := func(ctx context.Context, key string, opts ...cache.CallOption) (string, error) {
		_ = ctx
		n := loaderCalls.Add(1)
		time.Sleep(250 * time.Millisecond)
		v := fmt.Sprintf("value-v%d", n)
		fmt.Printf("loader called for key=%s -> %s\n", key, v)
		return v, nil
	}

	builder, err := cachalot.NewBuilder[string]("logic-expire-cache", store)
	if err != nil {
		panic(err)
	}
	cache, err := builder.
		WithLogicExpireDefaultLogicTTL(8 * minute).
		WithLogicExpireDefaultWriteBackTTL(10 * minute).
		WithLogicExpireLoader(loadFn).
		WithCacheMissLoader(loadFn).
		WithCacheMissDefaultWriteBackTTL(10 * minute).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("T=0m: set initial value")
	if err := cache.Set(ctx, "article:1", "seed-v0", 10*minute, store_ristretto.WithSynchronousSet(true)); err != nil {
		panic(err)
	}

	time.Sleep(8*minute + 100*time.Millisecond)
	fmt.Println("T=8m: concurrent reads after logical expiry (returns old value + triggers refresh)")

	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			v, err := cache.Get(ctx, "article:1")
			if err != nil {
				panic(err)
			}
			fmt.Printf("  reader-%d got: %s\n", id, v)
		}(i)
	}
	wg.Wait()

	fmt.Printf("loader call count after T=8m burst: %d\n", loaderCalls.Load())
	fresh, err := cache.Get(ctx, "article:1")
	if err != nil {
		panic(err)
	}
	fmt.Printf("after refresh, next get returns: %s\n", fresh)

	time.Sleep(2 * minute)
	fmt.Println("T=10m: delete key (simulate physical removal step in timeline)")
	if err := cache.Delete(ctx, "article:1"); err != nil {
		panic(err)
	}

	v, err := cache.Get(ctx, "article:1")
	if err != nil {
		panic(err)
	}
	fmt.Printf("after delete, miss-loader reloads value: %s\n", v)
	fmt.Printf("total loader calls: %d\n", loaderCalls.Load())
}
