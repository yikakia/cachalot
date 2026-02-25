package multicache

import (
	"context"
	"time"

	"github.com/sourcegraph/conc/pool"
	"github.com/yikakia/cachalot/core/cache"
)

// 返回 err 后，行为由 ErrorHandleMode 控制
type WriteBackFn[T any] func(ctx context.Context, getCtx *FetchContext[T], caches []cache.Cache[T]) error

// 并行执行，全部执行完才会返回
func WriteBackParallel[T any](defaultTTl time.Duration) WriteBackFn[T] {
	return func(ctx context.Context, getCtx *FetchContext[T], caches []cache.Cache[T]) error {
		p := &pool.ErrorPool{}
		for _, cache := range caches {
			cache := cache
			p.Go(func() error {
				return cache.Set(ctx, getCtx.Key, getCtx.GotValue, defaultTTl)
			})

		}
		return p.Wait()
	}
}
