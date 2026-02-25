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

	// 1) Create a local Ristretto store, then build a typed cache on top of it.
	client, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	store := store_ristretto.New(client, store_ristretto.WithStoreName("basic-ristretto"))

	builder, err := cachalot.NewBuilder[string]("basic-cache", store)
	if err != nil {
		panic(err)
	}
	c, err := builder.Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("[1] Set key=user:1")
	err = c.Set(ctx, "user:1", "Alice", 5*time.Minute, store_ristretto.WithSynchronousSet(true))
	if err != nil {
		panic(err)
	}

	fmt.Println("[2] Get hit (user:1)")
	val, err := c.Get(ctx, "user:1")
	if err != nil {
		panic(err)
	}
	fmt.Printf("    hit value: %q\n", val)

	fmt.Println("[3] Get miss (user:404)")
	_, err = c.Get(ctx, "user:404")
	if errors.Is(err, cachecore.ErrNotFound) {
		fmt.Println("    miss: key not found")
	} else if err != nil {
		panic(err)
	}

	fmt.Println("[4] Delete user:1 and verify miss")
	if err := c.Delete(ctx, "user:1"); err != nil {
		panic(err)
	}
	_, err = c.Get(ctx, "user:1")
	if errors.Is(err, cachecore.ErrNotFound) {
		fmt.Println("    delete success: key not found after delete")
	} else if err != nil {
		panic(err)
	}
}
