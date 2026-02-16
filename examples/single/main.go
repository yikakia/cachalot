package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot"
	store_ristretto "github.com/yikakia/cachalot/stores/ristretto"
)

func main() {
	// build client
	client, _ := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	// build store
	risStore := store_ristretto.New(client, store_ristretto.WithStoreName("localCache_ristretto"))

	// build cache
	builder, err := cachalot.NewBuilder[User]("a single cache", risStore)
	if err != nil {
		panic(err)
	}
	cache, err := builder.
		WithCacheMissLoader(getUser).                             // if key not exist then load from loaderFunc
		WithCacheMissDefaultWriteBackTTL(time.Millisecond * 300). // when get from loaderFunc, set val to the cache with ttl
		Build()
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	for range 5 {
		wg.Go(func() {
			u, err := cache.Get(context.Background(), "1", store_ristretto.WithSynchronousSet(true))
			if err != nil {
				panic(err)
			}
			fmt.Println("get:", u)
		})
	}

	wg.Wait()
	time.Sleep(time.Second) // wait for expire

	u, err := cache.Get(context.Background(), "1")
	if err != nil {
		panic(err)
	}
	fmt.Println("get after sleep:", u)
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
	fmt.Println("source called.", user)
	return user, nil
}
