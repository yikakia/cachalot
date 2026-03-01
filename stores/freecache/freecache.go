package freecache

import (
	"context"
	"fmt"
	"time"

	"github.com/coocood/freecache"
	"github.com/yikakia/cachalot/core/cache"
)

// Cache 定义 freecache 客户端需要实现的接口
type Cache interface {
	Get(key []byte) (value []byte, err error)
	GetWithExpiration(key []byte) (value []byte, expireAt uint32, err error)
	Set(key, value []byte, expireSeconds int) error
	Del(key []byte) (affected bool)
	Clear()
}

// New 创建一个新的 freecache 的 Store 封装
// client 是 freecache 的客户端实例
func New(client *freecache.Cache, opts ...Option) *Store {
	s := &Store{
		client: client,
		name:   "freecache",
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
	val, err := s.client.Get([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, fmt.Errorf("key:%s not found in store:%s. %w", key, s.name, cache.ErrNotFound)
		}
		return nil, err
	}
	return val, nil
}

func (s *Store) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (any, time.Duration, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}
	val, expireAt, err := s.client.GetWithExpiration([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, 0, fmt.Errorf("key:%s not found in store:%s. %w", key, s.name, cache.ErrNotFound)
		}
		return nil, 0, err
	}

	var ttl time.Duration
	if expireAt == 0 {
		// 永不过期
		ttl = 0
	} else {
		// 使用 time.Until 获取亚秒精度的剩余时间
		ttl = time.Until(time.Unix(int64(expireAt), 0))
		if ttl <= 0 {
			return nil, 0, fmt.Errorf("key:%s not found in store:%s. %w", key, s.name, cache.ErrNotFound)
		}
	}

	return val, ttl, nil
}

// Set 将值存入缓存
// val 必须是 []byte 类型
// ttl 只支持秒级的精度
func (s *Store) Set(ctx context.Context, key string, val any, ttl time.Duration, _ ...cache.CallOption) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if ttl < 0 {
		return cache.ErrInvalidTTL
	}

	b, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("freecache store only accepts []byte values, got %T", val)
	}

	expireSeconds := int(ttl.Seconds())
	return s.client.Set([]byte(key), b, expireSeconds)
}

// Delete 从缓存中删除指定 key
func (s *Store) Delete(ctx context.Context, key string, _ ...cache.CallOption) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.client.Del([]byte(key))
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
var _ Cache = (*freecache.Cache)(nil)
