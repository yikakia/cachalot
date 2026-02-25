package multicache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sourcegraph/conc/pool"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

// New 聚合多个 cache 与兜底的回源函数进行搭配使用
func New[T any](name string, cfg Config[T], caches ...cache.Cache[T]) (MultiCache[T], error) {
	m := &multiCache[T]{
		caches: caches,
		cfg:    &cfg,
	}

	var res MultiCache[T] = m
	if cfg.Observable != nil {
		res = newObservableDecorator(name, res, cfg.Observable)
	}

	return res, nil
}

// MultiCache  聚合多个 cache 并可以通过兜底函数进行
//
// 相比于 cache 接口 没有 GetWithTTL 方法 因为很难定义此时获取的 TTL 是哪个 cache 的。无论如何实现都可能造成误用 因此没有实现
type MultiCache[T any] interface {
	Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error)
	Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error
	Delete(ctx context.Context, key string, opts ...cache.CallOption) error
	Clear(ctx context.Context) error
	Caches() []cache.Cache[T]
	FetchByLoader(ctx context.Context, key string) (T, error)
	Logger() telemetry.Logger
	Metrics() telemetry.Metrics
}

type multiCache[T any] struct {
	caches []cache.Cache[T]
	cfg    *Config[T]
}

// Get 按照一定流程从传入的 cache 中查询，并自动回写
//
//	查询的流程如下
//	1. 通过传入的 FetchPolicy 获取 val，失败的 cache
//	2. 通过传入的 WriteBackCacheFilter 过滤出需要回写的失败的 cache
//	3. 通过传入的 WriteBackFn 进行回写
func (m *multiCache[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (val T, err error) {
	var zero T
	var getCtx = FetchContext[T]{
		Key:        key,
		Options:    opts,
		MultiCache: m,
	}
	val, failedCaches, err := m.cfg.FetchPolicy(ctx, &getCtx)
	if err != nil {
		return zero, err
	}
	getCtx.GotValue = val

	writeBackCaches := m.cfg.WriteBackCacheFilter(ctx, &getCtx, failedCaches)
	err = m.cfg.WriteBackFn(ctx, &getCtx, writeBackCaches)
	if err != nil {
		switch e := m.cfg.ErrorHandleMode; e {
		case ErrorHandleStrict:
			return zero, err
		case ErrorHandleTolerant:
			m.cfg.Observable.ErrorContext(ctx, "[multiCache] cache write back error", "key", key, "error", err.Error())
			return getCtx.GotValue, nil
		default:
			return zero, fmt.Errorf("[multiCache] unexpected error handling strategy: %v", e)
		}
	}
	return getCtx.GotValue, nil
}

// 并行执行，当出现错误时收集，最后全部返回
func (m *multiCache[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) (err error) {
	p := &pool.ErrorPool{}
	p.WithFirstError()
	for _, c := range m.caches {
		c := c
		p.Go(func() error {
			return c.Set(ctx, key, val, ttl, opts...)
		})
	}
	return p.Wait()
}

// 串行执行
func (m *multiCache[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) (err error) {
	var errs []error
	for _, c := range m.caches {
		err := c.Delete(ctx, key, opts...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// 串行执行
func (m *multiCache[T]) Clear(ctx context.Context) (err error) {
	var errs []error
	for _, cache := range m.caches {
		err := cache.Clear(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *multiCache[T]) Caches() []cache.Cache[T] {
	return m.caches
}

func (m *multiCache[T]) FetchByLoader(ctx context.Context, key string) (val T, err error) {
	return m.cfg.LoaderFn(ctx, key)
}

func (m *multiCache[T]) Logger() telemetry.Logger {
	return m.cfg.Observable.Logger
}

func (m *multiCache[T]) Metrics() telemetry.Metrics {
	return m.cfg.Observable.Metrics
}
