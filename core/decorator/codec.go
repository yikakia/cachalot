package decorator

import (
	"context"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/codec"
)

var _ cache.Cache[any] = (*CodecDecorator[any])(nil)

type CodecDecorator[T any] struct {
	cache.Cache[[]byte]
	Codec codec.Codec
}

func (t *CodecDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	var zero T
	get, err := t.Cache.Get(ctx, key, opts...)
	if err != nil {
		return zero, err
	}
	var target T
	err = t.Codec.Unmarshal(get, &target)
	if err != nil {
		return zero, err
	}

	return target, nil
}

func (t *CodecDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	marshal, err := t.Codec.Marshal(val)
	if err != nil {
		return err
	}

	return t.Cache.Set(ctx, key, marshal, ttl, opts...)
}

func (t *CodecDecorator[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	var zero T
	get, ttl, err := t.Cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return zero, 0, err
	}

	var target T
	err = t.Codec.Unmarshal(get, &target)
	if err != nil {
		return zero, 0, err
	}
	return target, ttl, nil
}
