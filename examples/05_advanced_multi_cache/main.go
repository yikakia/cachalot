package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/multicache"
	store_ristretto "github.com/yikakia/cachalot/stores/ristretto"
)

func newL1Cache() cache.Cache[string] {
	client, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	store := store_ristretto.New(client, store_ristretto.WithStoreName("L1-ristretto"))
	builder, err := cachalot.NewBuilder[string]("L1-cache", store)
	if err != nil {
		panic(err)
	}
	c, err := builder.Build()
	if err != nil {
		panic(err)
	}
	return c
}

// use slowMapStore not ristretto
// every call will delay 80*time.Millisecond
func newL2Cache() cache.Cache[string] {
	store := NewSlowMapStore("L2-slow-map", 80*time.Millisecond)
	builder, err := cachalot.NewBuilder[string]("L2-cache", store)
	if err != nil {
		panic(err)
	}
	c, err := builder.Build()
	if err != nil {
		panic(err)
	}
	return c
}

func timedGet(ctx context.Context, c multicache.MultiCache[string], key string) {
	start := time.Now()
	v, err := c.Get(ctx, key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("get %-12s -> %-18q (%s)\n", key, v, time.Since(start))
}

func main() {
	ctx := context.Background()
	l1 := newL1Cache()
	l2 := newL2Cache()

	if err := l2.Set(ctx, "only-l2", "value-from-L2", time.Minute); err != nil {
		panic(err)
	}

	var loaderCalls atomic.Int32
	loader := func(ctx context.Context, key string) (string, error) {
		_ = ctx
		n := loaderCalls.Add(1)
		time.Sleep(120 * time.Millisecond)
		v := fmt.Sprintf("loader-value-%d", n)
		fmt.Printf("loader called for key=%s -> %s\n", key, v)
		return v, nil
	}

	mc, err := cachalot.NewMultiBuilder("multi-L1-L2", l1, l2).
		WithLoader(loader).
		WithWriteBack(multicache.WriteBackParallel[string](time.Minute)).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("[1] L1 hit: fast return")
	if err := mc.Set(ctx, "hot", "value-in-L1", time.Minute, store_ristretto.WithSynchronousSet(true)); err != nil {
		panic(err)
	}
	timedGet(ctx, mc, "hot")

	fmt.Println("[2] L2 hit: first slower, then write-back makes second fast")
	timedGet(ctx, mc, "only-l2")
	timedGet(ctx, mc, "only-l2")

	fmt.Println("[3] Full miss: loader fetches once and writes back to both levels")
	timedGet(ctx, mc, "missing-key")
	timedGet(ctx, mc, "missing-key")
	fmt.Printf("loader calls: %d\n", loaderCalls.Load())
}
