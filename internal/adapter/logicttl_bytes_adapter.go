package adapter

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/decorator"
	"github.com/yikakia/cachalot/internal"
)

// LogicTTLBytesAdapter 将 LogicTTLValue[[]byte] 编码为 []byte：
// [8-byte little-endian UnixNano expireAt][raw payload]
type LogicTTLBytesAdapter[T any] struct {
	cache.Cache[[]byte]
}

func NewLogicTTLBytesAdapter[T any](next cache.Cache[[]byte]) (cache.Cache[decorator.LogicTTLValue[T]], error) {
	if !internal.IsBytesType[T]() {
		return nil, fmt.Errorf("logicTTLBytesAdapter only supports []byte value type, got %s", reflect.TypeFor[T]())
	}
	return &LogicTTLBytesAdapter[T]{Cache: next}, nil
}

func (l *LogicTTLBytesAdapter[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (decorator.LogicTTLValue[T], error) {
	raw, err := l.Cache.Get(ctx, key, opts...)
	if err != nil {
		return decorator.LogicTTLValue[T]{}, err
	}
	return decodeLogicTTLBytes[T](raw)
}

func (l *LogicTTLBytesAdapter[T]) Set(ctx context.Context, key string, val decorator.LogicTTLValue[T], ttl time.Duration, opts ...cache.CallOption) error {
	raw, err := encodeLogicTTLBytes(val)
	if err != nil {
		return err
	}
	return l.Cache.Set(ctx, key, raw, ttl, opts...)
}

func (l *LogicTTLBytesAdapter[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (decorator.LogicTTLValue[T], time.Duration, error) {
	raw, ttl, err := l.Cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return decorator.LogicTTLValue[T]{}, 0, err
	}
	decoded, err := decodeLogicTTLBytes[T](raw)
	if err != nil {
		return decorator.LogicTTLValue[T]{}, 0, err
	}
	return decoded, ttl, nil
}

func (l *LogicTTLBytesAdapter[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return l.Cache.Delete(ctx, key, opts...)
}

func (l *LogicTTLBytesAdapter[T]) Clear(ctx context.Context) error {
	return l.Cache.Clear(ctx)
}

func encodeLogicTTLBytes[T any](val decorator.LogicTTLValue[T]) ([]byte, error) {
	payload, ok := any(val.Val).([]byte)
	if !ok {
		return nil, fmt.Errorf("logic TTL value must be []byte, got %T", val.Val)
	}
	out := make([]byte, 8+len(payload))
	var ts int64
	if !val.ExpireAt.IsZero() {
		ts = val.ExpireAt.UnixNano()
	}
	binary.LittleEndian.PutUint64(out[:8], uint64(ts))
	copy(out[8:], payload)
	return out, nil
}

func decodeLogicTTLBytes[T any](raw []byte) (decorator.LogicTTLValue[T], error) {
	if len(raw) < 8 {
		return decorator.LogicTTLValue[T]{}, fmt.Errorf("invalid logic TTL bytes payload: len=%d", len(raw))
	}
	ts := int64(binary.LittleEndian.Uint64(raw[:8]))
	payload := raw[8:]
	typedPayload, ok := any(payload).(T)
	if !ok {
		return decorator.LogicTTLValue[T]{}, fmt.Errorf("internal type mismatch: expected %s from logic TTL payload", reflect.TypeFor[T]())
	}
	var expireAt time.Time
	if ts > 0 {
		expireAt = time.Unix(0, ts)
	}
	return decorator.LogicTTLValue[T]{
		Val:      typedPayload,
		ExpireAt: expireAt,
	}, nil
}
