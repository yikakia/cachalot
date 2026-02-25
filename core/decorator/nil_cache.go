package decorator

import (
	"context"
	"errors"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

var _ cache.Cache[any] = (*NilCacheDecorator[any])(nil)

// ProtectionFn 防止缓存击穿的函数签名
// 当缓存未命中时调用，返回一个默认值，该值会被写入缓存
type ProtectionFn[T any] func(key string) T

type NilCacheConfig[T any] struct {
	Cache        cache.Cache[T]
	ProtectionFn ProtectionFn[T]
	WriteBackTTL time.Duration
	Observer     *telemetry.Observable
}

func NewNilCacheDecorator[T any](config NilCacheConfig[T]) *NilCacheDecorator[T] {
	return &NilCacheDecorator[T]{
		cache:        config.Cache,
		protectionFn: config.ProtectionFn,
		writeBackTTL: config.WriteBackTTL,
		ob:           config.Observer,
	}
}

type NilCacheDecorator[T any] struct {
	cache        cache.Cache[T]
	protectionFn ProtectionFn[T]
	writeBackTTL time.Duration
	ob           *telemetry.Observable
}

func (d *NilCacheDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	val, err := d.cache.Get(ctx, key, opts...)
	if err == nil {
		return val, nil
	}

	if errors.Is(err, cache.ErrNotFound) && d.protectionFn != nil {
		return d.protectFromPenetration(ctx, key, opts...)
	}

	return val, err
}

func (d *NilCacheDecorator[T]) protectFromPenetration(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	// 调用防护函数获取防护值
	val := d.protectionFn(key)

	// 写回缓存
	err := d.Set(ctx, key, val, d.writeBackTTL, opts...)
	if err != nil {
		if d.ob != nil && d.ob.Logger != nil {
			d.ob.Logger.ErrorContext(ctx, "[NilCacheDecorator] write back failed.", "key", key, "err", err)
		}
	}

	return val, nil
}

func (d *NilCacheDecorator[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	val, ttl, err := d.cache.GetWithTTL(ctx, key, opts...)
	if err == nil {
		return val, ttl, nil
	}

	if errors.Is(err, cache.ErrNotFound) && d.protectionFn != nil {
		val, _ := d.protectFromPenetration(ctx, key, opts...)

		return val, d.writeBackTTL, nil
	}

	return val, ttl, err
}

func (d *NilCacheDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	return d.cache.Set(ctx, key, val, ttl, opts...)
}

func (d *NilCacheDecorator[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return d.cache.Delete(ctx, key, opts...)
}

func (d *NilCacheDecorator[T]) Clear(ctx context.Context) error {
	return d.cache.Clear(ctx)
}
