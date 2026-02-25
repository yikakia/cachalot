package decorator

import (
	"context"

	"golang.org/x/sync/singleflight"
)

type LoaderFn[T any] func(ctx context.Context, key string) (T, error)

func SingleflightWrapper[T any](fn LoaderFn[T]) LoaderFn[T] {
	g := &singleflight.Group{}
	return func(ctx context.Context, key string) (T, error) {
		var zero T
		ch := g.DoChan(key, func() (interface{}, error) {
			return fn(ctx, key)
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
