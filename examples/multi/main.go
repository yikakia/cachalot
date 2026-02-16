package main

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/multicache/write_back"
	"github.com/yikakia/cachalot/core/telemetry"
	store_ristretto "github.com/yikakia/cachalot/stores/ristretto"
)

func newCache(name string) cache.Cache[User] {
	// build client
	client, _ := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	// build store
	risStore := store_ristretto.New(client, store_ristretto.WithStoreName(name+"_localCache_ristretto"))

	// build cache
	builder, err := cachalot.NewBuilder[User](name, risStore)
	if err != nil {
		panic(err)
	}
	cache, err := builder.
		WithSingleFlight(true).
		WithMetrics(logMertic(name)).
		Build()
	if err != nil {
		panic(err)
	}
	return cache
}

type logMertic string

func (l logMertic) Record(ctx context.Context, evt *telemetry.Event) error {
	slog.Info(string(l+"_record_event"),
		slog.String("op", string(evt.Op)),
		slog.String("result", string(evt.Result)),
		slog.String("cacheName", evt.CacheName),
		slog.String("storeName", evt.StoreName),
		slog.Duration("latency", evt.Latency),
		slog.Any("customFields", evt.FrozenCustomFields()),
	)
	return nil
}

func main() {
	cache0 := newCache("cache0")
	cache1 := newCache("cache1")
	mulCache, err := cachalot.NewMultiBuilder("multiCacheName", cache0, cache1).
		WithWriteBack(write_back.Builder[User]{
			DefaultTTL: time.Second * 2,
		}.Build()).
		WithMetrics(logMertic("mulCache")).
		WithLoader(getUser).
		Build()

	wg := sync.WaitGroup{}
	for range 2 {
		wg.Go(func() {
			u, err := mulCache.Get(context.Background(), "1", store_ristretto.WithSynchronousSet(true))
			if err != nil {
				panic(err)
			}
			slog.Info("final_get", "get", u)
		})
	}

	wg.Wait()

	u, err := mulCache.Get(context.Background(), "1")
	if err != nil {
		panic(err)
	}
	slog.Info("after_wait_get", "get", u)
	// -- output --
	//source called. {1 name1 nickname1}
	//get: {1 name1 nickname1}
	//get: {1 name1 nickname1}
	//get: {1 name1 nickname1}
	//get: {1 name1 nickname1}
	//get: {1 name1 nickname1}
	//source called. {1 name1 nickname1}
	//get after sleep: {1 name1 nickname1}
}

type User struct {
	Id int

	Name     string
	NickName string
}

func getUser(ctx context.Context, key string) (User, error) {
	id, err := strconv.Atoi(key)
	if err != nil {
		return User{}, err
	}

	user := User{Id: id, Name: "name" + key, NickName: "nickname" + key}
	slog.Info("source called.", "key", key)
	return user, nil
}
