package decorator

import (
	"context"
	"errors"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

var _ cache.Cache[any] = (*MissedLoaderDecorator[any])(nil)

type MissedLoaderDecoratorConfig[T any] struct {
	Cache        cache.Cache[T]
	LoadFn       LoaderFn[T]
	WriteBackTTL time.Duration
	Observer     *telemetry.Observable
}

func NewMissedLoaderDecorator[T any](config MissedLoaderDecoratorConfig[T]) *MissedLoaderDecorator[T] {
	return &MissedLoaderDecorator[T]{
		cache:        config.Cache,
		loadFn:       config.LoadFn,
		writeBackTTL: config.WriteBackTTL,
		ob:           config.Observer,
	}
}

type MissedLoaderDecorator[T any] struct {
	cache        cache.Cache[T]
	loadFn       LoaderFn[T]
	writeBackTTL time.Duration
	ob           *telemetry.Observable
}

func (d *MissedLoaderDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	val, err := d.cache.Get(ctx, key, opts...)
	if err == nil {
		return val, nil
	}

	if errors.Is(err, cache.ErrNotFound) && d.loadFn != nil {
		return d.loadFromSource(ctx, key, opts...)
	}

	return val, err
}

func (d *MissedLoaderDecorator[T]) loadFromSource(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	var zero T
	val, err := d.loadFn(ctx, key, opts...)
	if err != nil {
		// load failed
		return zero, err
	}

	// write back
	err = d.Set(ctx, key, val, d.writeBackTTL, opts...)
	if err != nil {
		if d.ob != nil && d.ob.Logger != nil {
			d.ob.Logger.ErrorContext(ctx, "[MissedLoaderDecorator] write back failed.", "key", key, "err", err)
		}
	}

	return val, nil
}

func (d *MissedLoaderDecorator[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	val, ttl, err := d.cache.GetWithTTL(ctx, key, opts...)
	if err == nil {
		return val, ttl, nil
	}

	if errors.Is(err, cache.ErrNotFound) && d.loadFn != nil {
		val, err := d.loadFromSource(ctx, key, opts...)
		if err != nil {
			var zero T
			return zero, 0, err
		}
		return val, d.writeBackTTL, nil
	}

	return val, ttl, err
}

func (d *MissedLoaderDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	return d.cache.Set(ctx, key, val, ttl, opts...)
}

func (d *MissedLoaderDecorator[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return d.cache.Delete(ctx, key, opts...)
}

func (d *MissedLoaderDecorator[T]) Clear(ctx context.Context) error {
	return d.cache.Clear(ctx)
}
