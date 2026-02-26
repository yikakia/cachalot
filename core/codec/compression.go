package codec

import (
	"bytes"
	"compress/gzip"
	"io"
)

// GzipCompressionCodec 是默认压缩实现。
type GzipCompressionCodec struct {
	Level int
}

func (g GzipCompressionCodec) Compress(src []byte) ([]byte, error) {
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

func (g GzipCompressionCodec) Decompress(src []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}
