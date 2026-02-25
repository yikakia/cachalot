package cache

import (
	"github.com/yikakia/cachalot/core/telemetry"
)

type Configs[T any] struct {
	Error      error
	ob         *telemetry.Observable
	factory    CacheFactory[T]
	decorators []Decorator[T]
}

// OptionFn[T] defines configuration options for Cache[T]
type OptionFn[T any] func(*Configs[T])

func (o OptionFn[T]) Apply(cfgs *Configs[T]) {
	o(cfgs)
}

type Option[T any] interface {
	Apply(*Configs[T])
}

func WithOptions[T any](opts ...Option[T]) Option[T] {
	return OptionFn[T](func(o *Configs[T]) {
		for _, opt := range opts {
			if o.Error != nil {
				break
			}
			opt.Apply(o)
		}
	})
}

func WithObservable[T any](ob *telemetry.Observable) Option[T] {
	return OptionFn[T](func(o *Configs[T]) {
		o.ob = ob
	})
}

type SimpleCacheFactory[T any] func(Store) (Cache[T], error)
type CacheFactory[T any] func(Store, *telemetry.Observable) (Cache[T], error)

func WithSimpleFactory[T any](factory SimpleCacheFactory[T]) Option[T] {
	return WithFactory(func(store Store, _ *telemetry.Observable) (Cache[T], error) {
		return factory(store)
	})
}

func WithFactory[T any](factory CacheFactory[T]) Option[T] {
	return OptionFn[T](func(c *Configs[T]) {
		c.factory = factory
	})
}

type SimpleDecorator[T any] func(cache Cache[T]) (Cache[T], error)

type Decorator[T any] func(cache Cache[T], ob *telemetry.Observable) (Cache[T], error)

func WithDecorator[T any](d Decorator[T]) Option[T] {
	return OptionFn[T](func(o *Configs[T]) {
		o.decorators = append(o.decorators, d)
	})
}

func WithSimpleDecorator[T any](decorator SimpleDecorator[T]) Option[T] {
	return WithDecorator(func(cache Cache[T], _ *telemetry.Observable) (Cache[T], error) {
		return decorator(cache)
	})
}
