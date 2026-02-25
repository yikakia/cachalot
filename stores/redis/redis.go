package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yikakia/cachalot/core/cache"
)

type Client interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	FlushDB(ctx context.Context) *redis.StatusCmd
}

func New(client Client, opts ...Option) *Store {
	s := &Store{
		client: client,
		name:   "redis",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type Option func(*Store)

func WithStoreName(name string) Option {
	return func(s *Store) {
		s.name = name
	}
}

type Store struct {
	client Client
	name   string
}

func (s *Store) Get(ctx context.Context, key string, _ ...cache.CallOption) (any, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("key:%s not found in store:%s. %w", key, s.name, cache.ErrNotFound)
		}
		return nil, err
	}
	return val, nil
}

func (s *Store) Set(ctx context.Context, key string, val any, ttl time.Duration, _ ...cache.CallOption) error {
	if ttl < 0 {
		return cache.ErrInvalidTTL
	}
	raw, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("want:[]byte got:%T %w", val, cache.ErrTypeMissMatch)
	}
	return s.client.Set(ctx, key, raw, ttl).Err()
}

func (s *Store) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (any, time.Duration, error) {
	val, err := s.Get(ctx, key, opts...)
	if err != nil {
		return nil, 0, err
	}

	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return nil, 0, err
	}

	if ttl < 0 {
		return val, 0, nil
	}
	return val, ttl, nil
}

func (s *Store) Delete(ctx context.Context, key string, _ ...cache.CallOption) error {
	return s.client.Del(ctx, key).Err()
}

func (s *Store) Clear(ctx context.Context) error {
	return s.client.FlushDB(ctx).Err()
}

func (s *Store) StoreName() string {
	return s.name
}

var _ cache.Store = (*Store)(nil)
var _ Client = (*redis.Client)(nil)
