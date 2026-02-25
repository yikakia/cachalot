package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot"
	cachecore "github.com/yikakia/cachalot/core/cache"
	store_ristretto "github.com/yikakia/cachalot/stores/ristretto"
)

func main() {
	ctx := context.Background()

	client, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	store := store_ristretto.New(client, store_ristretto.WithStoreName("obs-ristretto"))

	logger := &consoleLogger{}
	metrics := newStatsMetrics()

	builder, err := cachalot.NewBuilder[string]("obs-cache", store)
	if err != nil {
		panic(err)
	}
	cache, err := builder.
		WithLogger(logger).
		WithMetrics(metrics).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("[1] Set")
	if err := cache.Set(ctx, "k1", "hello", time.Minute, store_ristretto.WithSynchronousSet(true)); err != nil {
		panic(err)
	}

	fmt.Println("[2] Get hit")
	if _, err := cache.Get(ctx, "k1"); err != nil {
		panic(err)
	}

	fmt.Println("[3] Get miss")
	_, err = cache.Get(ctx, "k404")
	if err != nil && !errors.Is(err, cachecore.ErrNotFound) {
		panic(err)
	}

	fmt.Println("[4] Delete (metrics intentionally returns an error to show custom logger output)")
	if err := cache.Delete(ctx, "k1"); err != nil {
		panic(err)
	}

	fmt.Println("[5] Metrics summary")
	metrics.Report()
}
