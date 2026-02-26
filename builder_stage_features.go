package cachalot

import (
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/decorator"
	"github.com/yikakia/cachalot/core/telemetry"
)

// CompressionCodec 定义字节压缩与解压能力。
type CompressionCodec interface {
	Compress(src []byte) ([]byte, error)
	Decompress(src []byte) ([]byte, error)
}

// ByteTransform 用于构建 []byte 层能力链（压缩/加密/签名等）。
type ByteTransform func(next cache.Cache[[]byte], ob *telemetry.Observable) (cache.Cache[[]byte], error)

// TypeAdapter 负责 T <-> []byte 的类型转换。
type TypeAdapter[T any] func(next cache.Cache[[]byte], ob *telemetry.Observable) (cache.Cache[T], error)

// WithCompression 以 Decorator 风格声明压缩能力，但内部会编译到 byte-stage。
func (b *Builder[T]) WithCompression(c CompressionCodec) *Builder[T] {
	return b.WithByteTransforms(func(next cache.Cache[[]byte], _ *telemetry.Observable) (cache.Cache[[]byte], error) {
		return decorator.NewCompressionDecorator(next, c), nil
	})
}

// WithByteTransforms 追加字节级转换链（按声明顺序执行）。
func (b *Builder[T]) WithByteTransforms(ts ...ByteTransform) *Builder[T] {
	b.features.byteTransforms = append(b.features.byteTransforms, ts...)
	return b
}

// WithTypeAdapter 覆盖默认类型适配器。
func (b *Builder[T]) WithTypeAdapter(a TypeAdapter[T]) *Builder[T] {
	b.features.typeAdapter = a
	return b
}
