package decorator

import (
	"context"

	"github.com/yikakia/cachalot/core/cache"
	"golang.org/x/sync/singleflight"
)

type LoaderFn[T any] func(ctx context.Context, key string, opts ...cache.CallOption) (T, error)

func SingleflightWrapper[T any](fn LoaderFn[T]) LoaderFn[T] {
	g := &singleflight.Group{}
	return func(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
		var zero T
		ch := g.DoChan(key, func() (interface{}, error) {
			return fn(ctx, key, opts...)
		})
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case res := <-ch:
			if res.Err != nil {
				return zero, res.Err
			}
			return res.Val.(T), nil
		}
	}
}
