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
	PTTL(ctx context.Context, key string) *redis.DurationCmd
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
	// 对 redis client 而言，ttl == -1 表示保持 ttl 不变
	// ttl > 0 表示重新设置
	// ttl <=0 && ttl != -1 表示 永不过期
	// TODO REVIEW 实际使用
	if ttl < 0 {
		return cache.ErrInvalidTTL
	}

	raw, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("want:[]byte got:%T %w", val, cache.ErrTypeMissMatch)
	}

	return s.client.Set(ctx, key, raw, ttl).Err()
}

// TODO 可配置项：lua 脚本还是 顺序
func (s *Store) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (any, time.Duration, error) {
	val, err := s.Get(ctx, key, opts...)
	if err != nil {
		return nil, 0, err
	}

	// TODO 或许可以支持配置使用 TTL 还是 PTTL
	ttl, err := s.client.PTTL(ctx, key).Result()
	if err != nil {
		return nil, 0, err
	}

	// TODO 已经拿到了 value 如果此时 ttl == -2 说明 key 不存在 或许应该报错
	// 如果此时 ttl == -1 说明 key 永不过期 但是这和版本相关 应该明确行为是什么
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
