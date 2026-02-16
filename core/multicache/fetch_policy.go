package multicache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

type FetchContext[T any] struct {
	Key     string
	Options []cache.CallOption

	Cache    MultiCache[T]
	GotValue T
}

// FetchPolicy 缓存获取策略（描述怎么从多个缓存中获取数据）
//
// 当返回 error 时，会直接返回 error 不会再触发回写的逻辑
type FetchPolicy[T any] func(ctx context.Context, getCtx *FetchContext[T]) (T, []FailedCache[T], error)
type FailedCache[T any] struct {
	Cache cache.Cache[T]
	Err   error
}

// 顺序遍历 cache 获取，source 兜底
func FetchPolicySequential[T any](ctx context.Context, getCtx *FetchContext[T]) (T, []FailedCache[T], error) {
	var zero T
	var failedCaches []FailedCache[T]
	tags := map[string]string{}
	defer telemetry.AddCustomFields(ctx, tags)

	m := getCtx.Cache
	key := getCtx.Key
	// 从缓存中加载
	for i, cache := range m.Caches() {
		val, err := cache.Get(ctx, key, getCtx.Options...)
		if err != nil {
			failedCaches = append(failedCaches, FailedCache[T]{
				Cache: cache,
				Err:   err,
			})
			continue
		}
		tags["source"] = "cache_" + strconv.Itoa(i)
		return val, failedCaches, nil
	}

	// 没有加载成功的，可能是不存在或者失败，这里认为是都需要回源
	val, err := m.FetchByLoader(ctx, key)
	if err != nil {
		// 回源失败了，这里直接返回 err
		return zero, nil, fmt.Errorf("[FetchPolicySequential] get from source failed: %w", err)
	}

	tags["source"] = "loader"
	return val, failedCaches, nil
}
