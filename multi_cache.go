package cachalot

import (
	"errors"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/multicache"
	"github.com/yikakia/cachalot/core/telemetry"
)

// MultiBuilder 用于构建 MultiCache
type MultiBuilder[T any] struct {
	err                  error
	name                 string
	caches               []cache.Cache[T]
	needLoaderFnNilCheck bool

	singleFlight bool
	metrics      telemetry.Metrics
	logger       telemetry.Logger
	cfg          multicache.Config[T]
}

// NewMultiBuilder 创建一个新的 MultiBuilder
// caches 是多级缓存的列表，优先级从前到后
// 默认配置时 需要使用 MultiBuilder.WithLoader 提供回源函数
//
// 默认配置为
//
// FetchPolicy = multicache.FetchPolicySequential 顺序遍历 cache 获取，source 兜底 需要调用 MultiBuilder.WithLoader 提供回源函数
// WriteBackCacheFilter = multicache.MissedCacheFilter 仅回源返回 cache.ErrNotFound 的 cache
// WriteBackFn = multicache.WriteBackParallel[T](time.Minute) 并行写回，同步等待，写回的ttl为一分钟
// ErrorHandleMode = multicache.ErrorHandleTolerant 宽容模式 当 WriteBackFn 返回 err 时，仅记录日志
func NewMultiBuilder[T any](name string, caches ...cache.Cache[T]) *MultiBuilder[T] {
	mb := &MultiBuilder[T]{
		name:                 name,
		caches:               caches,
		singleFlight:         true,
		needLoaderFnNilCheck: true,
		metrics:              telemetry.NoopMetrics(),
		logger:               telemetry.SlogLogger(),

		cfg: multicache.Config[T]{
			FetchPolicy:          multicache.FetchPolicySequential[T],
			WriteBackCacheFilter: multicache.MissedCacheFilter[T],
			WriteBackFn:          multicache.WriteBackParallel[T](time.Minute),
			ErrorHandleMode:      multicache.ErrorHandleTolerant,
		},
	}

	if name == "" {
		mb.err = errors.New("cache name cannot be empty")
	}
	return mb
}

// WithLoader 设置回源函数
// 默认配置 FetchPolicy = multicache.FetchPolicySequential 时必传 不然会运行时 panic
func (b *MultiBuilder[T]) WithLoader(fn multicache.LoaderFn[T]) *MultiBuilder[T] {
	b.cfg.LoaderFn = fn
	return b
}

// WithFetchPolicy 设置查询策略
func (b *MultiBuilder[T]) WithFetchPolicy(policy multicache.FetchPolicy[T]) *MultiBuilder[T] {
	b.cfg.FetchPolicy = policy
	return b
}

// WithWriteBack 设置回写策略
func (b *MultiBuilder[T]) WithWriteBack(fn multicache.WriteBackFn[T]) *MultiBuilder[T] {
	b.cfg.WriteBackFn = fn
	return b
}

// WithWriteBackFilter 设置回写过滤策略
func (b *MultiBuilder[T]) WithWriteBackFilter(filter multicache.WriteBackCacheFilter[T]) *MultiBuilder[T] {
	b.cfg.WriteBackCacheFilter = filter
	return b
}

// WithErrorHandling 设置错误处理模式
func (b *MultiBuilder[T]) WithErrorHandling(mode multicache.ErrorHandleMode) *MultiBuilder[T] {
	b.cfg.ErrorHandleMode = mode
	return b
}

// WithSingleflightLoader LoaderFn 的 singleflight 封装 默认开启
func (b *MultiBuilder[T]) WithSingleflightLoader(enabled bool) *MultiBuilder[T] {
	b.singleFlight = enabled
	return b
}

func (b *MultiBuilder[T]) WithLogger(logger telemetry.Logger) *MultiBuilder[T] {
	b.logger = logger
	return b
}

func (b *MultiBuilder[T]) WithMetrics(metrics telemetry.Metrics) *MultiBuilder[T] {
	b.metrics = metrics
	return b
}

// 如果使用的策略不需要 LoaderFn 则可以通过 MultiBuilder.WithLoaderFnNotNil(false) 禁止 LoaderFn 的 nil 检查
func (b *MultiBuilder[T]) WithLoaderFnNotNil(enabled bool) *MultiBuilder[T] {
	b.needLoaderFnNilCheck = enabled
	return b
}

// Build 构建 MultiCache
func (b *MultiBuilder[T]) Build() (multicache.MultiCache[T], error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.needLoaderFnNilCheck && b.cfg.LoaderFn == nil {
		return nil, errors.New("loader function is required")
	}

	finalCfg := b.cfg
	if !b.singleFlight {
		finalCfg.LoaderFn = multicache.SingleflightWrapper(finalCfg.LoaderFn)
	}

	finalCfg.Observable = &telemetry.Observable{
		Metrics: b.metrics,
		Logger:  b.logger,
	}

	return multicache.New(b.name, finalCfg, b.caches...)
}
