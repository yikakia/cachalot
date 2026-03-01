package valkey

import (
	"time"
)

type Option func(*Store)

func WithName(name string) Option {
	return func(store *Store) {
		store.name = name
	}
}

func WithClientSideCacheExpiration(duration time.Duration) Option {
	return func(store *Store) {
		store.clientSideCacheExpiration = duration
	}
}
