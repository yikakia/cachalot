package multicache

import (
	"context"
	"errors"

	"github.com/yikakia/cachalot/core/cache"
)

// WriteBackCacheFilter 回写缓存筛选（筛选需要回写的缓存）
type WriteBackCacheFilter[T any] func(ctx context.Context, getCtx *FetchContext[T], failedCaches []FailedCache[T]) []cache.Cache[T]

// 缓存不存在的才需要回源
func MissedCacheFilter[T any](_ context.Context, _ *FetchContext[T], failedCaches []FailedCache[T]) []cache.Cache[T] {
	var ret []cache.Cache[T]
	for _, failedCache := range failedCaches {
		if errors.Is(failedCache.Err, cache.ErrNotFound) {
			ret = append(ret, failedCache.Cache)
		}
	}
	return ret
}
