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
	"github.com/yikakia/cachalot/internal"
	"github.com/yikakia/cachalot/internal/adapter"
)

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
			d, err := decorator.NewLogicTTLDecorator(cfg)
			if err != nil {
				return nil, err
			}
			return d, nil
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

	return b.adaptBytesToLogicWire(byteCache)
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
	if internal.IsBytesType[T]() {
		return newBytesPassThroughCache[T](next), nil
	}
	return nil, fmt.Errorf("byte-stage enabled but no adapter configured for type %s: configure WithCodec or WithTypeAdapter", reflect.TypeFor[T]().String())
}

func (b *Builder[T]) adaptBytesToLogicWire(next cache.Cache[[]byte]) (cache.Cache[decorator.LogicTTLValue[T]], error) {
	if b.features.codec != nil {
		return newCodecDecorator[decorator.LogicTTLValue[T]](next, b.features.codec), nil
	}
	// 内置支持 T=[]byte 的逻辑过期 wire 适配，避免强制依赖 codec。
	if internal.IsBytesType[T]() {
		if b.features.logicExpire.enableBytesAdapter {
			return adapter.NewLogicTTLBytesAdapter[T](next)
		}
		return nil, fmt.Errorf("logicTTLBytesAdapter supports []byte value type. try use builder.WithLogicExpireBytesAdapter(true) to enable it, or configure WithCodec for LogicTTLValue[T] <-> []byte adapter")
	}

	return nil, fmt.Errorf("logic-expire byte-stage requires adapter for %s: configure WithCodec for LogicTTLValue[T] <-> []byte", reflect.TypeFor[T]().String())
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
