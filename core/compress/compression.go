package compress

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"io"
)

// GzipCompression 是基于标准库 gzip 的压缩实现。
type GzipCompression struct {
	Level int
}

func (g GzipCompression) Compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	level := g.Level
	if level == 0 {
		level = gzip.DefaultCompression
	}
	zw, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err = zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g GzipCompression) Decompress(src []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

// ZlibCompression 是基于标准库 zlib 的压缩实现。
type ZlibCompression struct {
	Level int
}

func (z ZlibCompression) Compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	level := z.Level
	if level == 0 {
		level = zlib.DefaultCompression
	}
	zw, err := zlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err = zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (z ZlibCompression) Decompress(src []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

// FlateCompression 是基于标准库 flate 的压缩实现。
type FlateCompression struct {
	Level int
}

func (f FlateCompression) Compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	level := f.Level
	if level == 0 {
		level = flate.DefaultCompression
	}
	zw, err := flate.NewWriter(&buf, level)
	if err != nil {
		return nil, err
	}
	if _, err = zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (f FlateCompression) Decompress(src []byte) ([]byte, error) {
	zr := flate.NewReader(bytes.NewReader(src))
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}

// LZWCompression 是基于标准库 lzw 的压缩实现。
// 默认参数为 LSB + 8bit literal width，和 GIF 中常见配置一致。
type LZWCompression struct {
	Order        lzw.Order
	LiteralWidth int
}

func (l LZWCompression) Compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer

	order := l.Order
	if order != lzw.LSB && order != lzw.MSB {
		order = lzw.LSB
	}

	literalWidth := l.LiteralWidth
	if literalWidth == 0 {
		literalWidth = 8
	}

	zw := lzw.NewWriter(&buf, order, literalWidth)
	if _, err := zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (l LZWCompression) Decompress(src []byte) ([]byte, error) {
	order := l.Order
	if order != lzw.LSB && order != lzw.MSB {
		order = lzw.LSB
	}

	literalWidth := l.LiteralWidth
	if literalWidth == 0 {
		literalWidth = 8
	}

	zr := lzw.NewReader(bytes.NewReader(src), order, literalWidth)
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}
