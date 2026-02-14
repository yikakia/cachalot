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

	// 当启用逻辑过期和codec任一特性时 如果再调用 factory 则会报错

	// codec 特有的配置项
	codec codec.Codec

	// 逻辑过期特有的配置项
	logicTTLEnabled bool
	defaultLogicTTL time.Duration
	writeBackTTL    time.Duration

	// 回源函数 非必须
	loadFn decorator.LoadFn[T]
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
	b.buildFactory()
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

func (b *Builder[T]) appendErr(err error) {
	b.err = errors.Join(b.err, err)
}

func (b *Builder[T]) buildFactory() {
	// 不开启任何特性
	if !b.features.logicTTLEnabled && b.features.codec == nil {
		// 兜底配置
		if b.factory == nil {
			b.factory = cache.WithSimpleFactory(func(store cache.Store) (cache.Cache[T], error) {
				return cache.NewBaseCache[T](store), nil
			})
		}
		return
	}

	// 现在是多 feature 的组合 如果已经有了实现 此时应该再次报错兜底
	if b.factoryCustomized {
		b.appendErr(fmt.Errorf("[Builder.buildFactory] already initialized while feature ttlEnabled:%v, codecEnabled:%v", b.features.logicTTLEnabled, b.features.codec != nil))
		return
	}

	switch {
	case b.features.logicTTLEnabled && b.features.codec == nil:
		// 仅logic
		b.factory = cache.WithFactory(func(store cache.Store, ob *telemetry.Observable) (cache.Cache[T], error) {
			c := cache.NewBaseCache[decorator.LogicTTLValue[T]](store)
			cfg := b.buildTTLConfig(c, ob)
			return decorator.NewLogicTTLDecorator(cfg), nil
		})
		return
	case b.features.logicTTLEnabled && b.features.codec != nil:
		// logic && codec
		b.factory = cache.WithFactory(func(store cache.Store, ob *telemetry.Observable) (cache.Cache[T], error) {
			base := cache.NewBaseCache[[]byte](store)
			_codec := newCodecDecorator[decorator.LogicTTLValue[T]](base, b.features.codec)
			cfg := b.buildTTLConfig(_codec, ob)
			return decorator.NewLogicTTLDecorator(cfg), nil
		})
	case b.features.codec != nil:
		b.factory = cache.WithFactory(func(store cache.Store, ob *telemetry.Observable) (cache.Cache[T], error) {
			base := cache.NewBaseCache[[]byte](store)
			return newCodecDecorator[T](base, b.features.codec), nil
		})
	default:
		// 这个分支应该永远走不到
		panic("unreachable for Builder.buildFactory")
	}
}

func (b *Builder[T]) buildTTLConfig(next cache.Cache[decorator.LogicTTLValue[T]], ob *telemetry.Observable) decorator.LogicTTLDecoratorConfig[T] {
	// 校验并配置默认值
	d := decorator.LogicTTLDecoratorConfig[T]{
		Cache:           next,
		DefaultLogicTTL: time.Minute,
		LoadFn:          b.features.loadFn,
		WriteBackTTL:    2 * time.Minute,
		Observer:        ob,
	}

	if b.features.defaultLogicTTL > 0 {
		d.DefaultLogicTTL = b.features.defaultLogicTTL
	}
	if b.features.writeBackTTL > 0 {
		d.WriteBackTTL = b.features.writeBackTTL
	}

	return d
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

func newCodecDecorator[T any](next cache.Cache[[]byte], codec codec.Codec) *decorator.CodecDecorator[T] {
	return &decorator.CodecDecorator[T]{
		Cache: next,
		Codec: codec,
	}
}
