package decorator

import (
	"context"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

var _ cache.Cache[any] = (*LogicTTLDecorator[any])(nil)

type LogicTTLMetrics interface {
	RecordLogicExpire(ctx context.Context)
}

type LoadFn[T any] func(ctx context.Context, key string) (T, error)
type LogicTTLDecoratorConfig[T any] struct {
	Cache           cache.Cache[LogicTTLValue[T]]
	DefaultLogicTTL time.Duration
	// 当逻辑过期后，提供回源函数
	LoadFn LoadFn[T]
	// 当设置了回源函数时，回写时的物理过期时间
	WriteBackTTL time.Duration

	Observer *telemetry.Observable
}

func NewLogicTTLDecorator[T any](config LogicTTLDecoratorConfig[T]) *LogicTTLDecorator[T] {
	l := &LogicTTLDecorator[T]{
		cache:           config.Cache,
		ob:              config.Observer,
		defaultLogicTTL: config.DefaultLogicTTL,
		loadFn:          config.LoadFn,
		writeBackTTL:    config.WriteBackTTL,
	}
	if ttlMetrics, ok := config.Observer.Metrics.(LogicTTLMetrics); ok {
		l.logicExpireMetrics = ttlMetrics.RecordLogicExpire
	}

	return l
}

type LogicTTLDecorator[T any] struct {
	cache              cache.Cache[LogicTTLValue[T]]
	ob                 *telemetry.Observable
	logicExpireMetrics func(ctx context.Context)

	defaultLogicTTL time.Duration
	loadFn          func(ctx context.Context, key string) (T, error)
	writeBackTTL    time.Duration
}

func (d *LogicTTLDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	var zero T
	val, err := d.cache.Get(ctx, key, opts...)
	if err != nil {
		return zero, err
	}

	if val.IsExpire() {

		d.onExpire(ctx, key, opts...)

	}

	return val.Val, nil
}

func (d *LogicTTLDecorator[T]) onExpire(ctx context.Context, key string, opts ...cache.CallOption) {
	if d.logicExpireMetrics != nil {
		d.logicExpireMetrics(ctx)
	}

	val, err := d.loadFn(ctx, key)
	if err != nil {
		d.ob.Logger.ErrorContext(ctx, "[LogicTTLDecorator] load from source failed.", "key", key, "err", err)
	}

	err = d.Set(ctx, key, val, d.writeBackTTL, opts...)
	if err != nil {
		d.ob.Logger.ErrorContext(ctx, "[LogicTTLDecorator] write back failed.", "key", key, "err", err)
	}
}

func (d *LogicTTLDecorator[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	var zero T
	val, ttl, err := d.cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return zero, 0, err
	}
	// 如果能获取 说明没有物理过期
	// todo 或许应该用一个 feature 开关，让调用方决定应该返回逻辑过期时间还是物理过期时间
	return val.Val, ttl, err
}

func (d *LogicTTLDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	var setOptions cache.CallOptConfig
	for _, opt := range opts {
		opt(&setOptions)
	}

	var expireAt time.Time
	if d.defaultLogicTTL > 0 {
		expireAt = time.Now().Add(d.defaultLogicTTL)
	}

	setVal := LogicTTLValue[T]{
		Val:      val,
		ExpireAt: expireAt,
	}

	return d.cache.Set(ctx, key, setVal, ttl, opts...)
}

func (d *LogicTTLDecorator[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return d.cache.Delete(ctx, key, opts...)
}

func (d *LogicTTLDecorator[T]) Clear(ctx context.Context) error {
	return d.cache.Clear(ctx)
}

type LogicTTLValue[T any] struct {
	Val      T
	ExpireAt time.Time
}

func (t *LogicTTLValue[T]) IsExpire() bool {
	return !t.ExpireAt.IsZero() && time.Now().After(t.ExpireAt)
}
