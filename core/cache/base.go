package cache

import (
	"context"
	"fmt"
	"time"
)

func NewBaseCache[T any](store Store) *BaseCache[T] {
	return &BaseCache[T]{store: store}
}

var _ Cache[[]byte] = &BaseCache[[]byte]{}

func baseCacheFactory[T any](s Store) Cache[T] {
	return &BaseCache[T]{store: s}
}

type BaseCache[T any] struct {
	store Store
}

func (w *BaseCache[T]) Get(ctx context.Context, key string, opts ...CallOption) (T, error) {
	var zero T
	val, err := w.store.Get(ctx, key, opts...)
	if err != nil {
		return zero, err
	}

	if v, ok := val.(T); ok {
		return v, nil
	}

	return zero, fmt.Errorf("[BaseCache]:want:%T got:%T %w", zero, val, ErrTypeMissMatch)
}

func (w *BaseCache[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...CallOption) error {
	return w.store.Set(ctx, key, val, ttl, opts...)
}

func (w *BaseCache[T]) GetWithTTL(ctx context.Context, key string, opts ...CallOption) (T, time.Duration, error) {
	var zero T
	val, ttl, err := w.store.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return zero, 0, err
	}

	if v, ok := val.(T); ok {
		return v, ttl, nil
	}

	return zero, 0, fmt.Errorf("[BaseCache]:want:%T got:%T %w", zero, val, ErrTypeMissMatch)
}

func (w *BaseCache[T]) Delete(ctx context.Context, key string, opts ...CallOption) error {
	return w.store.Delete(ctx, key, opts...)
}

func (w *BaseCache[T]) Clear(ctx context.Context) error {
	return w.store.Clear(ctx)
}
