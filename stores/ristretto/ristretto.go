package ristretto

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/yikakia/cachalot/core/cache"
)

// Client 定义 ristretto 客户端需要实现的接口
type Cache interface {
	Get(key string) (any, bool)
	GetTTL(key string) (time.Duration, bool)
	SetWithTTL(key string, value any, cost int64, ttl time.Duration) bool
	Del(key string)
	Wait()
	Clear()
}

// New 创建一个新的 ristretto 的 Store 封装
// client 是 ristretto 的客户端实例
func New(client *ristretto.Cache[string, any], opts ...Option) *Store {
	s := &Store{
		client: client,
		name:   "ristretto",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option 定义 Store 的选项函数
type Option func(*Store)

func WithStoreName(name string) Option {
	return func(s *Store) {
		s.name = name
	}
}

type Store struct {
	client Cache
	name   string
}

// Get 从缓存中获取值
func (s *Store) Get(ctx context.Context, key string, _ ...cache.CallOption) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	val, found := s.client.Get(key)
	if !found {
		return nil, fmt.Errorf("key:%s not found in store:%s. %w", key, s.name, cache.ErrNotFound)
	}
	return val, nil
}

func (s *Store) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (any, time.Duration, error) {
	val, err := s.Get(ctx, key, opts...)
	if err != nil {
		return nil, 0, err
	}
	// 如果过期了，那么在上面 Get 的时候就已经返回 err 了 所以这里一定是没过期的
	ttl, _ := s.client.GetTTL(key)
	return val, ttl, nil
}

// Set 将值存入缓存
func (s *Store) Set(ctx context.Context, key string, val any, ttl time.Duration, opts ...cache.CallOption) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if ttl < 0 {
		return cache.ErrInvalidTTL
	}
	setOpt := cache.ApplyOptions(opts...)
	features := loadOrInitSetFeatures(setOpt)

	cost := int64(1)
	if features.cost > 1 {
		cost = features.cost
	}
	// cost 表示每个条目占用相同的成本
	s.client.SetWithTTL(key, val, cost, ttl)

	if features.flush {
		s.client.Wait()
	}

	return nil
}

// Delete 从缓存中删除指定 key
func (s *Store) Delete(ctx context.Context, key string, _ ...cache.CallOption) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.client.Del(key)
	return nil
}

// Clear 清空所有缓存
func (s *Store) Clear(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.client.Clear()
	return nil
}

// StoreName 返回 Store 名称
func (s *Store) StoreName() string {
	return s.name
}

var _ cache.Store = (*Store)(nil)
var _ Cache = (*ristretto.Cache[string, any])(nil)
