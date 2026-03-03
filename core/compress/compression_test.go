package compress

import (
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGzipCompression_RoundTrip(t *testing.T) {
	c := GzipCompression{Level: gzip.BestCompression}
	assertRoundTrip(t, c)
}

func TestGzipCompression_DefaultLevel(t *testing.T) {
	c := GzipCompression{}
	assertRoundTrip(t, c)
}

func TestGzipCompression_DecompressInvalidData(t *testing.T) {
	c := GzipCompression{}
	_, err := c.Decompress([]byte("invalid gzip payload"))
	require.Error(t, err)
}

func TestZlibCompression_RoundTrip(t *testing.T) {
	c := ZlibCompression{Level: zlib.BestCompression}
	assertRoundTrip(t, c)
}

func TestZlibCompression_DefaultLevel(t *testing.T) {
	c := ZlibCompression{}
	assertRoundTrip(t, c)
}

func TestZlibCompression_DecompressInvalidData(t *testing.T) {
	c := ZlibCompression{}
	_, err := c.Decompress([]byte("invalid zlib payload"))
	require.Error(t, err)
}

func TestFlateCompression_RoundTrip(t *testing.T) {
	c := FlateCompression{Level: flate.BestCompression}
	assertRoundTrip(t, c)
}

func TestFlateCompression_DefaultLevel(t *testing.T) {
	c := FlateCompression{}
	assertRoundTrip(t, c)
}

func TestFlateCompression_DecompressInvalidData(t *testing.T) {
	c := FlateCompression{}
	_, err := c.Decompress([]byte("invalid flate payload"))
	require.Error(t, err)
}

func TestLZWCompression_RoundTrip(t *testing.T) {
	c := LZWCompression{
		Order:        lzw.MSB,
		LiteralWidth: 8,
	}
	assertRoundTrip(t, c)
}

func TestLZWCompression_DefaultConfig(t *testing.T) {
	c := LZWCompression{}
	assertRoundTrip(t, c)
}

func TestLZWCompression_DecompressInvalidData(t *testing.T) {
	c := LZWCompression{}
	_, err := c.Decompress([]byte("invalid lzw payload"))
	require.Error(t, err)
}

func assertRoundTrip(t *testing.T, c interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
}) {
	t.Helper()

	plain := []byte("hello compression codec roundtrip hello compression codec roundtrip")
	compressed, err := c.Compress(plain)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	got, err := c.Decompress(compressed)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}
