# cachalot

The cache facade for Golang, aims to be developer friendly

[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/yikakia/cachalot?tab=doc)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yikakia/cachalot)

## Overview

* Single-level caching with pluggable storage backends
* Multi-level caching with configurable fetch and write-back policies
* Feature composition through a decorator pattern
* Comprehensive telemetry with structured logging and metrics
* Type safety using Go generics throughout the API
* High-level API for general users to use tested features (serialization, logical TTL, singleflight) through builder pattern while keeping the extensibility
* Low-level API for advanced users to compose their own cache through decorator pattern while leveraging the built-in features
* Developer Friendly


## Quick Start

see more [examples](examples)
### single level cache
```go
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
	// build the client
	client, _ := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1 << 10,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	// build the store
	risStore := store_ristretto.New(client, store_ristretto.WithStoreName("localCache_ristretto"))

	// build the cache
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

```

### multicache
```go


```
## License

© yikakia, 2026~time.Now()

Released under the [MIT License](https://github.com/yikakia/cachalot/blob/master/LICENSE)