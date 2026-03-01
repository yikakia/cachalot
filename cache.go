package cachalot

import (
	"errors"
	"fmt"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/codec"
	"github.com/yikakia/cachalot/core/decorator"
	"github.com/yikakia/cachalot/core/telemetry"
	"golang.org/x/sync/singleflight"
)

type features[T any] struct {
	// enabled by default
	singleFlight bool

	// codec 特有的配置项（默认类型适配器）
	codec codec.Codec
	// 用户自定义类型适配器（T <-> []byte）
	typeAdapter TypeAdapter[T]
	// 字节级转换链，例如压缩/加密
	byteTransforms []ByteTransform

	// 逻辑过期特有的配置项
	logicExpire struct {
		enabled bool
		// 默认十分钟逻辑过期
		defaultLogicTTL time.Duration
		// 回源函数 非必须
		loadFn decorator.LoaderFn[T]
		// 回源后写回缓存 默认一小时的物理过期时间
		defaultWriteBackTTL time.Duration
		// 如果 T = []byte，是否启用内置的 LogicTTLBytesAdapter 适配器，避免强制依赖 codec 包 默认不开启
		enableBytesAdapter bool
	}

	// cacheMissLoader 独有的配置
	missLoader struct {
		// 缓存过期后如何回源
		loadFn decorator.LoaderFn[T]
		// 回源后写回缓存，默认一小时过期
		defaultWriteBackTTL time.Duration
	}

	// 防缓存击穿功能配置
	nilCache struct {
		protectionFn decorator.ProtectionFn[T]
		// 防护值写回缓存时的TTL 默认一小时
		defaultWriteBackTTL time.Duration
	}
}

func NewBuilder[T any](name string, store cache.Store, opts ...cache.Option[T]) (*Builder[T], error) {
	if name == "" {
		return nil, errors.New("cache name is required")
	}
	if store == nil {
		return nil, errors.New("cache store is required")
	}

	b := &Builder[T]{
		cacheName: name,
		store:     store,
		features: features[T]{
			singleFlight: true,
		},
		factory: cache.WithSimpleFactory(func(store cache.Store) (cache.Cache[T], error) {
			return cache.NewBaseCache[T](store), nil
		}),
		obDecorators: cache.WithDecorator[T](func(cache cache.Cache[T], ob *telemetry.Observable) (cache.Cache[T], error) {
			return decorator.NewObservableDecorator(cache, store.StoreName(), name, ob), nil
		}),
		metrics: telemetry.NoopMetrics(),
		logger:  telemetry.SlogLogger(),
		options: opts,
	}

	b.features.logicExpire.defaultLogicTTL = 10 * time.Minute
	b.features.logicExpire.defaultWriteBackTTL = time.Hour

	b.features.missLoader.defaultWriteBackTTL = time.Hour

	b.features.nilCache.defaultWriteBackTTL = time.Hour

	return b, nil
}

type Builder[T any] struct {
	err error

	// cacheName
	cacheName string
	store     cache.Store

	// cache.BaseCache by default
	factory cache.Option[T]
	// telemetry.NoopMetrics by default
	metrics telemetry.Metrics
	// telemetry.SlogLogger by default
	logger telemetry.Logger
	// decorator.NewObservableDecorator by default
	obDecorators cache.Option[T]

	features   features[T]
	decorators []cache.Option[T]
	options    []cache.Option[T]

	factoryCustomized bool
}

func (b *Builder[T]) Build() (cache.Cache[T], error) {
	b.compileStages()
	b.decorateCacheMissedLoader()
	b.decoratePenetrationProtection()
	b.decorateSingleflight()
	if b.err != nil {
		return nil, fmt.Errorf("builder configs wrong: %w", b.err)
	}

	c, err := cache.New[T](b.cacheName, b.store,
		cache.WithObservable[T](&telemetry.Observable{
			Metrics: b.metrics,
			Logger:  b.logger,
		}),
		b.factory,
		cache.WithOptions(b.decorators...),
		b.obDecorators,
		cache.WithOptions(b.options...),
	)

	if err != nil {
		return nil, fmt.Errorf("build cache [%s] failed: %w", b.cacheName, err)
	}
	return c, nil
}

// 在 missedLoader 装饰器之后注入防缓存击穿装饰器
func (b *Builder[T]) decoratePenetrationProtection() {
	if b.features.nilCache.protectionFn == nil {
		return
	}
	writeBackTTL := b.features.nilCache.defaultWriteBackTTL
	if writeBackTTL < 0 {
		b.appendErr(fmt.Errorf("nilCache.writeBackTTL require >= 0, but got: %v", writeBackTTL))
		return
	}
	b.decorators = append(b.decorators, cache.WithDecorator(func(c cache.Cache[T], ob *telemetry.Observable) (cache.Cache[T], error) {
		return decorator.NewNilCacheDecorator(decorator.NilCacheConfig[T]{
			Cache:        c,
			ProtectionFn: b.features.nilCache.protectionFn,
			WriteBackTTL: writeBackTTL,
			Observer:     ob,
		}), nil
	}))
}

func (b *Builder[T]) appendErr(err error) {
	b.err = errors.Join(b.err, err)
}

// 需要在 []decorator 最后一层 在最后 build 的时候才可以调用
func (b *Builder[T]) decorateSingleflight() {
	if b.features.singleFlight {
		// 插入最后
		b.decorators = append(b.decorators, cache.WithDecorator(func(cache cache.Cache[T], ob *telemetry.Observable) (cache.Cache[T], error) {
			return &decorator.SingleflightDecorator[T]{
				Cache: cache,
				Group: &singleflight.Group{},
			}, nil
		}))
	}
}

func (b *Builder[T]) decorateCacheMissedLoader() {
	loadFn := b.features.missLoader.loadFn
	if loadFn == nil {
		return
	}

	writeBackTTL := b.features.missLoader.defaultWriteBackTTL
	if writeBackTTL < 0 {
		b.appendErr(fmt.Errorf("writeBackTTL require >= 0, but got: %v", writeBackTTL))
		return
	}

	wrappedFn := decorator.SingleflightWrapper[T](loadFn)
	b.decorators = append(b.decorators, cache.WithDecorator(func(c cache.Cache[T], ob *telemetry.Observable) (cache.Cache[T], error) {

		return decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[T]{
			Cache:        c,
			LoadFn:       wrappedFn,
			WriteBackTTL: writeBackTTL,
			Observer:     ob,
		}), nil
	}))
}
