package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot"
	"github.com/yikakia/cachalot/core/codec"
	store_ristretto "github.com/yikakia/cachalot/stores/ristretto"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

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
	store := store_ristretto.New(client, store_ristretto.WithStoreName("codec-ristretto"))

	jsonCodec := codec.JSONCodec{}
	builder, err := cachalot.NewBuilder[User]("user-cache", store)
	if err != nil {
		panic(err)
	}
	cache, err := builder.
		WithCodec(jsonCodec).
		Build()
	if err != nil {
		panic(err)
	}

	user := User{ID: 7, Name: "Alice", Email: "alice@example.com"}
	fmt.Printf("Original object: %+v\n", user)

	// 1) Show the raw JSON process explicitly.
	encoded, err := jsonCodec.Marshal(user)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Marshal(User) -> JSON bytes: %s\n", string(encoded))

	var decoded User
	if err := jsonCodec.Unmarshal(encoded, &decoded); err != nil {
		panic(err)
	}
	fmt.Printf("Unmarshal(JSON) -> User: %+v\n", decoded)

	// 2) Cache the object. Codec decorator will store []byte in the underlying store.
	key := "user:7"
	if err := cache.Set(ctx, key, user, 5*time.Minute, store_ristretto.WithSynchronousSet(true)); err != nil {
		panic(err)
	}
	client.Wait()

	// 3) Read raw payload from store to prove what is physically stored.
	raw, err := store.Get(ctx, key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Underlying store value type: %T\n", raw)
	fmt.Printf("Underlying store raw bytes: %s\n", string(raw.([]byte)))

	// 4) Read from cache and recover to typed struct automatically.
	cachedUser, err := cache.Get(ctx, key)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Cache.Get -> restored object: %+v\n", cachedUser)
}
