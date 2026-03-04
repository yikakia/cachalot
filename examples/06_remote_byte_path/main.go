package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yikakia/cachalot"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/codec"
	"github.com/yikakia/cachalot/core/compress"
	"github.com/yikakia/cachalot/core/decorator"
	store_redis "github.com/yikakia/cachalot/stores/redis"
)

type UserProfile struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Tier      string    `json:"tier"`
	UpdatedAt time.Time `json:"updated_at"`
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("missing required env %s", key))
	}
	return v
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("invalid int env %s=%q: %v", key, v, err))
	}
	return n
}

func main() {
	ctx := context.Background()

	// This example is Redis-only (no in-memory fallback).
	// Required:
	//   REDIS_ADDR=127.0.0.1:6379
	// Optional:
	//   REDIS_PASSWORD=
	//   REDIS_DB=0
	addr := mustEnv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")
	db := envInt("REDIS_DB", 0)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	defer func() { _ = rdb.Close() }()
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Errorf("redis ping failed: %w", err))
	}

	store := store_redis.New(rdb, store_redis.WithStoreName("remote-redis"))
	jsonCodec := codec.JSONCodec{}
	gzipCodec := compress.GzipCompression{}

	var sourceCalls atomic.Int32
	loadFromSource := func(ctx context.Context, key string, opts ...cache.CallOption) (UserProfile, error) {
		_ = opts
		n := sourceCalls.Add(1)
		time.Sleep(150 * time.Millisecond)
		v := UserProfile{
			ID:        101,
			Name:      fmt.Sprintf("Alice-v%d", n),
			Tier:      "premium",
			UpdatedAt: time.Now(),
		}
		fmt.Printf("source loader called key=%s -> %+v\n", key, v)
		return v, nil
	}

	cacheKey := "profile:101"
	builder, err := cachalot.NewBuilder[UserProfile]("remote-byte-path", store)
	if err != nil {
		panic(err)
	}
	c, err := builder.
		WithSingleflight(true).
		WithCodec(jsonCodec).
		WithCompression(gzipCodec).
		WithLogicExpireDefaultLogicTTL(2 * time.Second).
		WithLogicExpireDefaultWriteBackTTL(20 * time.Second).
		WithLogicExpireLoader(loadFromSource).
		WithCacheMissLoader(loadFromSource).
		WithCacheMissDefaultWriteBackTTL(20 * time.Second).
		Build()
	if err != nil {
		panic(err)
	}

	if err := c.Delete(ctx, cacheKey); err != nil {
		panic(err)
	}

	fmt.Println("[1] first get: cache miss -> load from source -> write back to redis")
	first, err := c.Get(ctx, cacheKey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("cache get #1: %+v\n", first)

	fmt.Println("[2] inspect raw bytes in redis and decode byte-path payload")
	raw, err := store.Get(ctx, cacheKey)
	if err != nil {
		panic(err)
	}
	rawBytes := raw.([]byte)
	fmt.Printf("redis value type=%T, raw size=%d bytes\n", rawBytes, len(rawBytes))

	decodedBytes, err := gzipCodec.Decompress(rawBytes)
	if err != nil {
		panic(err)
	}
	var wire decorator.LogicTTLValue[UserProfile]
	if err := jsonCodec.Unmarshal(decodedBytes, &wire); err != nil {
		panic(err)
	}
	fmt.Printf("byte path decode -> logic expire at=%s payload=%+v\n", wire.ExpireAt.Format(time.RFC3339Nano), wire.Val)

	fmt.Println("[3] wait for logical expiry, then concurrent reads return stale data while refresh runs once")
	time.Sleep(2300 * time.Millisecond)

	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			v, err := c.Get(ctx, cacheKey)
			if err != nil {
				panic(err)
			}
			fmt.Printf("reader-%d got stale-or-fresh value: name=%s updated_at=%s\n", id, v.Name, v.UpdatedAt.Format(time.RFC3339Nano))
		}(i)
	}
	wg.Wait()

	// give async refresh a short window to finish write-back
	time.Sleep(300 * time.Millisecond)
	latest, err := c.Get(ctx, cacheKey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("[4] post-refresh get: %+v\n", latest)
	fmt.Printf("source loader call count: %d\n", sourceCalls.Load())

	fmt.Println("done")
}
