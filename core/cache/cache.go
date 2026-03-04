package cache

import (
	"context"
	"time"
)

// New 创建一个带类型的 Cache[T]
// 它内部包裹了一个 Store，并在 Get/Set 时进行类型断言/转换
//
// 调用时至少应传入 observable factory store 用于构建基本的缓存
// name 参数保留用于可观测性/元数据扩展，当前不参与构建流程。
func New[T any](name string, store Store, opts ...Option[T]) (Cache[T], error) {
	cfg := &Configs[T]{}
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	if cfg.Error != nil {
		return nil, cfg.Error
	}

	// cache decorator
	c, err := cfg.factory(store, cfg.ob)
	if err != nil {
		return nil, err
	}

	for _, decorator := range cfg.decorators {
		c, err = decorator(c, cfg.ob)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Cache 用户操作的接口 封装了对存储层的操作逻辑
// T 是缓存值的类型
type Cache[T any] interface {
	Get(ctx context.Context, key string, opts ...CallOption) (T, error)
	Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...CallOption) error
	GetWithTTL(ctx context.Context, key string, opts ...CallOption) (T, time.Duration, error)
	Delete(ctx context.Context, key string, opts ...CallOption) error
	Clear(ctx context.Context) error
}
