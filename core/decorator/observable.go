package decorator

import (
	"context"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
	"github.com/yikakia/cachalot/internal"
)

func NewObservableDecorator[T any](
	cache cache.Cache[T],
	storeName string,
	cacheName string,
	ob *telemetry.Observable) *ObservableDecorator[T] {

	return &ObservableDecorator[T]{
		ob:        ob,
		storeName: storeName,
		cacheName: cacheName,
		cache:     cache,
	}
}

type ObservableDecorator[T any] struct {
	ob        *telemetry.Observable
	storeName string
	cacheName string
	cache     cache.Cache[T]
}

func (o *ObservableDecorator[T]) initCtx(ctx context.Context, op telemetry.Op) (context.Context, *telemetry.Event) {
	evt := &telemetry.Event{
		Op:        op,
		CacheName: o.cacheName,
		StoreName: o.storeName,
	}
	return telemetry.ContextWithEvent(ctx, evt), evt
}

func (o *ObservableDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (_ T, finalErr error) {
	start := time.Now()
	ctx, evt := o.initCtx(ctx, telemetry.OpGet)
	defer func() {
		evt.Error = finalErr
		evt.Latency = time.Since(start)
		evt.Result = internal.ResultFromErr(finalErr)

		err := o.ob.Metrics.Record(ctx, evt)
		if err != nil {
			o.ob.Logger.ErrorContext(ctx, "[ObservableDecorator.Get] Record Metrics Failed.", "err", err.Error())
		}
	}()

	var zero T
	val, err := o.cache.Get(ctx, key, opts...)
	if err != nil {
		return zero, err
	}
	return val, nil
}

func (o *ObservableDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) (finalErr error) {
	start := time.Now()
	ctx, evt := o.initCtx(ctx, telemetry.OpSet)
	defer func() {
		evt.Error = finalErr
		evt.Latency = time.Since(start)

		err := o.ob.Metrics.Record(ctx, evt)
		if err != nil {
			o.ob.Logger.ErrorContext(ctx, "[ObservableDecorator.Set] Record Metrics Failed.", "err", err.Error())
		}
	}()

	err := o.cache.Set(ctx, key, val, ttl)
	if err != nil {
		return err
	}
	return nil
}

func (o *ObservableDecorator[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (_ T, _ time.Duration, finalErr error) {
	start := time.Now()
	ctx, evt := o.initCtx(ctx, telemetry.OpGetWithTTL)
	defer func() {
		evt.Error = finalErr
		evt.Latency = time.Since(start)
		evt.Result = internal.ResultFromErr(finalErr)

		err := o.ob.Metrics.Record(ctx, evt)
		if err != nil {
			o.ob.Logger.ErrorContext(ctx, "[ObservableDecorator.GetWithTTL] Record Metrics Failed.", "err", err.Error())
		}
	}()

	var zero T
	val, ttl, err := o.cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return zero, 0, err
	}
	return val, ttl, nil
}

func (o *ObservableDecorator[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) (finalErr error) {
	start := time.Now()
	ctx, evt := o.initCtx(ctx, telemetry.OpDelete)
	defer func() {
		evt.Error = finalErr
		evt.Latency = time.Since(start)

		err := o.ob.Metrics.Record(ctx, evt)
		if err != nil {
			o.ob.Logger.ErrorContext(ctx, "[ObservableDecorator.Delete] Record Metrics Failed.", "err", err.Error())
		}
	}()

	return o.cache.Delete(ctx, key, opts...)
}

func (o *ObservableDecorator[T]) Clear(ctx context.Context) (finalErr error) {
	start := time.Now()
	ctx, evt := o.initCtx(ctx, telemetry.OpClear)
	defer func() {
		evt.Error = finalErr
		evt.Latency = time.Since(start)

		err := o.ob.Metrics.Record(ctx, evt)
		if err != nil {
			o.ob.Logger.ErrorContext(ctx, "[ObservableDecorator.Clear] Record Metrics Failed.", "err", err.Error())
		}
	}()

	return o.cache.Clear(ctx)
}

var _ cache.Cache[any] = (*ObservableDecorator[any])(nil)
