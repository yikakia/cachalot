package multicache

import (
	"context"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
	"github.com/yikakia/cachalot/internal"
)

type observableDecorator[T any] struct {
	MultiCache[T]
	ob   *telemetry.Observable
	name string
}

func newObservableDecorator[T any](name string, inner MultiCache[T], ob *telemetry.Observable) MultiCache[T] {
	return &observableDecorator[T]{
		MultiCache: inner,
		ob:         ob,
		name:       name,
	}
}

func (d *observableDecorator[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (val T, err error) {
	startTime := time.Now()
	var evt = &telemetry.Event{
		Op:        telemetry.OpGet,
		CacheName: d.name,
	}
	defer func() {
		evt.Result = internal.ResultFromErr(err)
		evt.Error = err
		evt.Latency = time.Since(startTime)
		_ = d.ob.Metrics.Record(ctx, evt)
	}()
	ctx = telemetry.ContextWithEvent(ctx, evt)

	return d.MultiCache.Get(ctx, key, opts...)
}

func (d *observableDecorator[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) (err error) {
	startTime := time.Now()
	var evt = &telemetry.Event{
		Op:        telemetry.OpSet,
		CacheName: d.name,
	}
	defer func() {
		evt.Error = err
		evt.Latency = time.Since(startTime)
		_ = d.ob.Metrics.Record(ctx, evt)
	}()
	ctx = telemetry.ContextWithEvent(ctx, evt)

	return d.MultiCache.Set(ctx, key, val, ttl, opts...)
}

func (d *observableDecorator[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) (err error) {
	startTime := time.Now()
	var evt = &telemetry.Event{
		Op:        telemetry.OpDelete,
		CacheName: d.name,
	}
	defer func() {
		evt.Error = err
		evt.Latency = time.Since(startTime)
		_ = d.ob.Metrics.Record(ctx, evt)
	}()
	ctx = telemetry.ContextWithEvent(ctx, evt)

	return d.MultiCache.Delete(ctx, key, opts...)
}

func (d *observableDecorator[T]) Clear(ctx context.Context) (err error) {
	startTime := time.Now()
	var evt = &telemetry.Event{
		Op:        telemetry.OpClear,
		CacheName: d.name,
	}
	defer func() {
		evt.Error = err
		evt.Latency = time.Since(startTime)
		_ = d.ob.Metrics.Record(ctx, evt)
	}()
	ctx = telemetry.ContextWithEvent(ctx, evt)

	return d.MultiCache.Clear(ctx)
}

func (d *observableDecorator[T]) FetchByLoader(ctx context.Context, key string) (val T, err error) {
	startTime := time.Now()
	var evt = &telemetry.Event{
		Op:        "fetch_by_loader",
		CacheName: d.name,
	}
	defer func() {
		evt.Result = internal.ResultFromErr(err)
		evt.Error = err
		evt.Latency = time.Since(startTime)
		_ = d.ob.Metrics.Record(ctx, evt)
	}()
	ctx = telemetry.ContextWithEvent(ctx, evt)

	return d.MultiCache.FetchByLoader(ctx, key)
}

func (d *observableDecorator[T]) Logger() telemetry.Logger {
	return d.ob.Logger
}

func (d *observableDecorator[T]) Metrics() telemetry.Metrics {
	return d.ob.Metrics
}
