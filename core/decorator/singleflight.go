package decorator

import (
	"context"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
	"golang.org/x/sync/singleflight"
)

var _ cache.Cache[any] = (*SingleflightDecorator[any])(nil)

// SingleflightDecorator[T] 使用 singleflight 包装 Cache[T] 的 Get 操作
// 如果启用了观测，则会在 Get GetWithTTL 中注入
// shared  true,false 标明该请求是否是shared
type SingleflightDecorator[T any] struct {
	Cache cache.Cache[T]
	Group *singleflight.Group
}

func (s *SingleflightDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	val, err, shared := s.Group.Do(key, func() (any, error) {
		return s.Cache.Get(ctx, key, opts...)
	})
	s.addTags(ctx, shared)
	if err != nil {
		var zero T
		return zero, err
	}
	return val.(T), nil
}

func (s *SingleflightDecorator[T]) addTags(ctx context.Context, shared bool) {
	tags := map[string]string{
		"shared": "true",
	}
	if shared {
		tags["shared"] = "false"
	}

	telemetry.AddCustomFields(ctx, tags)
}

func (s *SingleflightDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	return s.Cache.Set(ctx, key, val, ttl, opts...)
}

type withTTL[T any] struct {
	V   T
	TTL time.Duration
}

func (s *SingleflightDecorator[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	var zero T
	valAny, err, shared := s.Group.Do(key, func() (any, error) {
		value, ttl, err := s.Cache.GetWithTTL(ctx, key, opts...)
		if err != nil {
			return nil, err
		}
		return withTTL[T]{value, ttl}, nil
	})
	s.addTags(ctx, shared)
	if err != nil {
		return zero, 0, err
	}
	val := valAny.(withTTL[T])
	return val.V, val.TTL, nil
}

func (s *SingleflightDecorator[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return s.Cache.Delete(ctx, key, opts...)
}

func (s *SingleflightDecorator[T]) Clear(ctx context.Context) error {
	return s.Cache.Clear(ctx)
}
