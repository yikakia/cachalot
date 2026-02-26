package cachalot

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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
			missLoader: struct {
				loadFn              decorator.LoaderFn[T]
				defaultWriteBackTTL time.Duration
			}{
				loadFn:              nil,
				defaultWriteBackTTL: 0,
			},
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

func (b *Builder[T]) compileStages() {
	if b.factoryCustomized {
		if b.hasStagedFeaturesEnabled() {
			b.appendErr(errors.New("WithFactory cannot be combined with staged features (codec/logic-expire/compression/type-adapter), use WithCustomPlan or disable staged features"))
		}
		return
	}

	if b.features.logicExpire.enabled && !b.checkTTLConfigValid() {
		return
	}

	if b.features.logicExpire.enabled {
		b.factory = cache.WithFactory(func(store cache.Store, ob *telemetry.Observable) (cache.Cache[T], error) {
			wire, err := b.buildLogicWireCache(store, ob)
			if err != nil {
				return nil, err
			}
			cfg := b.buildTTLConfig(wire, ob)
			return decorator.NewLogicTTLDecorator(cfg), nil
		})
		return
	}

	b.factory = cache.WithFactory(func(store cache.Store, ob *telemetry.Observable) (cache.Cache[T], error) {
		return b.buildPlainTypedCache(store, ob)
	})
}

func (b *Builder[T]) hasStagedFeaturesEnabled() bool {
	return b.features.logicExpire.enabled ||
		b.features.codec != nil ||
		b.features.typeAdapter != nil ||
		len(b.features.byteTransforms) > 0
}

func (b *Builder[T]) buildPlainTypedCache(store cache.Store, ob *telemetry.Observable) (cache.Cache[T], error) {
	if !b.requiresBytePath() {
		return cache.NewBaseCache[T](store), nil
	}

	byteCache, err := b.buildByteCache(store, ob)
	if err != nil {
		return nil, err
	}

	return b.adaptBytesToType(byteCache, ob)
}

func (b *Builder[T]) buildLogicWireCache(store cache.Store, ob *telemetry.Observable) (cache.Cache[decorator.LogicTTLValue[T]], error) {
	if !b.requiresBytePath() {
		return cache.NewBaseCache[decorator.LogicTTLValue[T]](store), nil
	}

	byteCache, err := b.buildByteCache(store, ob)
	if err != nil {
		return nil, err
	}

	// 逻辑过期的 wire type 是 LogicTTLValue[T]，当前仅 codec 能做该泛型适配。
	if b.features.codec == nil {
		return nil, errors.New("logic-expire with byte transforms requires codec to adapt LogicTTLValue[T] <-> []byte")
	}

	return newCodecDecorator[decorator.LogicTTLValue[T]](byteCache, b.features.codec), nil
}

func (b *Builder[T]) buildByteCache(store cache.Store, ob *telemetry.Observable) (cache.Cache[[]byte], error) {
	var current cache.Cache[[]byte] = cache.NewBaseCache[[]byte](store)
	var err error
	for _, transform := range b.features.byteTransforms {
		current, err = transform(current, ob)
		if err != nil {
			return nil, err
		}
	}
	return current, nil
}

func (b *Builder[T]) adaptBytesToType(next cache.Cache[[]byte], ob *telemetry.Observable) (cache.Cache[T], error) {
	if b.features.typeAdapter != nil {
		return b.features.typeAdapter(next, ob)
	}
	if b.features.codec != nil {
		return newCodecDecorator[T](next, b.features.codec), nil
	}
	if isBytesType[T]() {
		return newBytesPassThroughCache[T](next), nil
	}
	return nil, fmt.Errorf("byte-stage enabled but no adapter configured for type %s: configure WithCodec or WithTypeAdapter", reflect.TypeFor[T]().String())
}

func (b *Builder[T]) requiresBytePath() bool {
	if len(b.features.byteTransforms) > 0 {
		return true
	}
	if b.features.codec != nil {
		return true
	}
	if b.features.typeAdapter != nil {
		return true
	}
	return false
}

func isBytesType[T any]() bool {
	return reflect.TypeFor[T]() == reflect.TypeFor[[]byte]()
}

func (b *Builder[T]) checkTTLConfigValid() bool {
	if ttl := b.features.logicExpire.defaultLogicTTL; ttl < 0 {
		b.appendErr(fmt.Errorf("logicExpire.defaultLogicTTL require >= 0 but got:%v", ttl))
		return false
	}
	if ttl := b.features.logicExpire.defaultWriteBackTTL; ttl < 0 {
		b.appendErr(fmt.Errorf("logicExpire.defaultWriteBackTTL require >= 0 but got:%v", ttl))
		return false
	}

	return true
}

func (b *Builder[T]) buildTTLConfig(next cache.Cache[decorator.LogicTTLValue[T]], ob *telemetry.Observable) decorator.LogicTTLDecoratorConfig[T] {
	loadFn := b.features.logicExpire.loadFn
	if loadFn != nil {
		loadFn = decorator.SingleflightWrapper(loadFn)
	}
	// 校验并配置默认值
	d := decorator.LogicTTLDecoratorConfig[T]{
		Cache:           next,
		DefaultLogicTTL: b.features.logicExpire.defaultLogicTTL,
		LoadFn:          loadFn,
		WriteBackTTL:    b.features.logicExpire.defaultWriteBackTTL,
		Observer:        ob,
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

func newCodecDecorator[T any](next cache.Cache[[]byte], codec codec.Codec) *decorator.CodecDecorator[T] {
	return &decorator.CodecDecorator[T]{
		Cache: next,
		Codec: codec,
	}
}

type bytesPassThroughCache[T any] struct {
	cache.Cache[[]byte]
}

func newBytesPassThroughCache[T any](next cache.Cache[[]byte]) cache.Cache[T] {
	return &bytesPassThroughCache[T]{Cache: next}
}

func (c *bytesPassThroughCache[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	var zero T
	v, err := c.Cache.Get(ctx, key, opts...)
	if err != nil {
		return zero, err
	}
	typed, ok := any(v).(T)
	if !ok {
		return zero, fmt.Errorf("internal type mismatch: expected %s from []byte bridge", reflect.TypeFor[T]())
	}
	return typed, nil
}

func (c *bytesPassThroughCache[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	raw, ok := any(val).([]byte)
	if !ok {
		return fmt.Errorf("internal type mismatch: expected %s to be []byte", reflect.TypeFor[T]())
	}
	return c.Cache.Set(ctx, key, raw, ttl, opts...)
}

func (c *bytesPassThroughCache[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	var zero T
	v, ttl, err := c.Cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return zero, 0, err
	}
	typed, ok := any(v).(T)
	if !ok {
		return zero, 0, fmt.Errorf("internal type mismatch: expected %s from []byte bridge", reflect.TypeFor[T]())
	}
	return typed, ttl, nil
}
