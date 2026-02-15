package cachalot

import (
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
func (b *Builder[T]) WithLogicTTLEnabled(enabled bool) *Builder[T] {
	b.features.logicTTLEnabled = enabled
	return b
}

// 默认设置的逻辑过期时间
func (b *Builder[T]) WithDefaultLogicTTL(d time.Duration) *Builder[T] {
	b.features.logicTTLEnabled = true
	b.features.defaultLogicTTL = d
	return b
}

// 逻辑过期回源后，回写时的物理过期时间
func (b *Builder[T]) WithWriteBackTTL(d time.Duration) *Builder[T] {
	b.features.logicTTLEnabled = true
	b.features.writeBackTTL = d
	return b
}

// 逻辑过期和缓存不存在加载都可能调用
// 需要手动使用 WithLogicTTLEnabled(true) 启用该功能
func (b *Builder[T]) WithLoadFn(fn decorator.LoadFn[T]) *Builder[T] {
	b.features.loadFn = fn
	return b
}

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
// 如果传入 cache.WithFactory 则会覆盖 ttl 和 codec 相关特性
// 如果传入 cache.WithObservable 则会覆盖 WithLogger 和 WithMetrics 的特性开启
// 如果传入 cache.WithDecorator 则会在 WithDecorators 的外层，观测层的内层
func (b *Builder[T]) WithOptions(opts ...cache.Option[T]) *Builder[T] {
	b.options = append(b.options, opts...)
	return b
}
