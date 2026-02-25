package cache

import (
	"context"
	"time"
)

// Store 存储层抽象，对接对象缓存 (ristretto) 字节缓存 (redis)
type Store interface {
	Get(ctx context.Context, key string, opts ...CallOption) (any, error)
	// 该键值将在 ttl 时间后过期，如果 ttl 为 0，表示永不过期，如果 ttl 为负数，会返回错误 ErrInvalidTTL
	Set(ctx context.Context, key string, val any, ttl time.Duration, opts ...CallOption) error
	// 如果永不过期，ttl 返回 0
	GetWithTTL(ctx context.Context, key string, opts ...CallOption) (any, time.Duration, error)
	Delete(ctx context.Context, key string, opts ...CallOption) error
	Clear(ctx context.Context) error
	StoreName() string
}

// 在对 client 直接进行封装时，不应实现额外的 feature， 例如 client 本身不接受 context.Context 时
// 就不要额外封装对 ctx.Done 感知的行为，如果要额外增加功能，应当对 StoreName 包裹一层显式地进行实现
