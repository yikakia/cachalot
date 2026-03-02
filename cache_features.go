package cachalot

import (
	"fmt"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/codec"
	"github.com/yikakia/cachalot/core/decorator"
	"github.com/yikakia/cachalot/core/telemetry"
)

// 给 Get GetWithTTL 开启 singleflight 功能
func (b *Builder[T]) WithSingleFlight(enable bool) *Builder[T] {
	b.features.singleFlight = enable
	return b
}

// 开启 Codec 功能
func (b *Builder[T]) WithCodec(codec codec.Codec) *Builder[T] {
	b.features.codec = codec
	return b
}

// 开启逻辑过期功能
func (b *Builder[T]) WithLogicExpireEnabled(enabled bool) *Builder[T] {
	b.features.logicExpire.enabled = enabled
	return b
}

// 默认设置的逻辑过期时间
// 需要大于等于0
func (b *Builder[T]) WithLogicExpireDefaultLogicTTL(d time.Duration) *Builder[T] {
	b.features.logicExpire.enabled = true
	b.features.logicExpire.defaultLogicTTL = d
	if d < 0 {
		b.appendErr(fmt.Errorf("logicExpireDefaultLogicTTL must >= 0, but got %v", d))
	}
	return b
}

// 逻辑过期回源后，回写时的物理过期时间
// 需要大于等于0
func (b *Builder[T]) WithLogicExpireDefaultWriteBackTTL(d time.Duration) *Builder[T] {
	b.features.logicExpire.enabled = true
	b.features.logicExpire.defaultWriteBackTTL = d
	if d < 0 {
		b.appendErr(fmt.Errorf("logicExpireDefaultWriteBackTTL must >= 0, but got: %v", d))
	}
	return b
}

// 如果 T = []byte，是否启用内置的 LogicTTLBytesAdapter 适配器
//
// 启用后会在逻辑过期功能的基础上，自动将 LogicTTLValue[[]byte] 编码为 []byte 存储，格式为 [8-byte little-endian UnixNano expireAt][raw payload]
// 使用场景: 源数据是 []byte 使用了压缩功能，同时还想使用逻辑过期的功能，可以启用这个适配器，避免引入 codec 包和额外的编码开销。
func (b *Builder[T]) WithLogicExpireBytesAdapter(enable bool) *Builder[T] {
	b.features.logicExpire.enableBytesAdapter = enable
	if enable {
		b.features.logicExpire.enabled = true
	}
	return b
}

func (b *Builder[T]) WithLogicExpireLoader(fn decorator.LoaderFn[T]) *Builder[T] {
	b.features.logicExpire.enabled = true
	b.features.logicExpire.loadFn = fn
	return b
}

func (b *Builder[T]) WithCacheMissLoader(fn decorator.LoaderFn[T]) *Builder[T] {
	b.features.missLoader.loadFn = fn
	return b
}

// 如果不调用 WithCacheMissLoader 传入回源函数的话 此设置无效
func (b *Builder[T]) WithCacheMissDefaultWriteBackTTL(d time.Duration) *Builder[T] {
	b.features.missLoader.defaultWriteBackTTL = d
	return b
}

// WithNilCacheFn 启用防缓存击穿功能
func (b *Builder[T]) WithNilCacheFn(fn decorator.ProtectionFn[T]) *Builder[T] {
	b.features.nilCache.protectionFn = fn
	return b
}

func (b *Builder[T]) WithNilCacheWriteBackTTL(ttl time.Duration) *Builder[T] {
	b.features.nilCache.defaultWriteBackTTL = ttl
	return b
}

// WithFactory 显式声明使用自定义装配计划，与 staged features 互斥。
func (b *Builder[T]) WithFactory(factory cache.CacheFactory[T]) *Builder[T] {
	b.factoryCustomized = true
	b.factory = cache.WithFactory(factory)
	return b
}

func (b *Builder[T]) WithLogger(logger telemetry.Logger) *Builder[T] {
	b.logger = logger
	return b
}

func (b *Builder[T]) WithMetrics(metrics telemetry.Metrics) *Builder[T] {
	b.metrics = metrics
	return b
}

// WithObserveDecorator 自定义最外层进行观测的 observer 装饰层
func (b *Builder[T]) WithObserveDecorator(d cache.Decorator[T]) *Builder[T] {
	b.obDecorators = cache.WithDecorator(d)
	return b
}

// WithDecorators 如果用 WithOptions 也传入了装饰器，则 WithOptions 传入的更靠外层
func (b *Builder[T]) WithDecorators(decorators ...cache.Decorator[T]) *Builder[T] {
	for _, d := range decorators {
		b.decorators = append(b.decorators, cache.WithDecorator(d))
	}
	return b
}

// WithOptions 自定义的可选项，优先级最高
//
// 如果传入 cache.WithFactory 则会覆盖 Builder 的装配计划
// 如果传入 cache.WithObservable 则会覆盖 WithLogger 和 WithMetrics 的特性开启
// 如果传入 cache.WithDecorator 则会在 WithDecorators 的外层，观测层的内层
func (b *Builder[T]) WithOptions(opts ...cache.Option[T]) *Builder[T] {
	b.options = append(b.options, opts...)
	return b
}
