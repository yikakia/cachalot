package decorator

import (
	"context"
	"time"

	"github.com/yikakia/cachalot/core/cache"
)

type CompressionCodec interface {
	Compress(src []byte) ([]byte, error)
	Decompress(src []byte) ([]byte, error)
}

var _ cache.Cache[[]byte] = (*CompressionDecorator)(nil)

type CompressionDecorator struct {
	cache.Cache[[]byte]
	codec CompressionCodec
}

func NewCompressionDecorator(next cache.Cache[[]byte], codec CompressionCodec) *CompressionDecorator {
	return &CompressionDecorator{
		Cache: next,
		codec: codec,
	}
}

func (d *CompressionDecorator) Get(ctx context.Context, key string, opts ...cache.CallOption) ([]byte, error) {
	raw, err := d.Cache.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}
	return d.codec.Decompress(raw)
}

func (d *CompressionDecorator) Set(ctx context.Context, key string, val []byte, ttl time.Duration, opts ...cache.CallOption) error {
	compressed, err := d.codec.Compress(val)
	if err != nil {
		return err
	}
	return d.Cache.Set(ctx, key, compressed, ttl, opts...)
}

func (d *CompressionDecorator) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) ([]byte, time.Duration, error) {
	raw, ttl, err := d.Cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return nil, 0, err
	}
	decoded, err := d.codec.Decompress(raw)
	if err != nil {
		return nil, 0, err
	}
	return decoded, ttl, nil
}
