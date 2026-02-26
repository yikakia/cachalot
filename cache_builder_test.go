package cachalot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/codec"
	"github.com/yikakia/cachalot/core/telemetry"
	"github.com/yikakia/cachalot/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewBuilderValidation(t *testing.T) {
	_, err := NewBuilder[string]("", nil)
	require.Error(t, err)

	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)

	_, err = NewBuilder[string]("cache", store)
	require.NoError(t, err)
}

func TestBuilderBuildAndDelegateToStore(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	store := mocks.NewMockStore(ctrl)
	store.EXPECT().StoreName().Return("mock-store").Times(1)
	store.EXPECT().Set(gomock.Any(), "k", "v", time.Minute).Return(nil)
	store.EXPECT().Get(gomock.Any(), "k").Return("v", nil)

	builder, err := NewBuilder[string]("single-cache", store)
	require.NoError(t, err)

	c, err := builder.Build()
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))
	v, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, "v", v)
}

func TestBuilderRejectsNegativeLogicTTL(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)
	builder, err := NewBuilder[string]("single-cache", store)
	require.NoError(t, err)

	_, err = builder.WithLogicExpireDefaultLogicTTL(-time.Second).Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "logicExpireDefaultLogicTTL")
}

func TestBuilderCompressionWithBytesWithoutCodec(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	store := mocks.NewMockStore(ctrl)

	plain := []byte("hello-compress")
	var written []byte

	store.EXPECT().StoreName().Return("mock-store").Times(1)
	store.EXPECT().Set(gomock.Any(), "k", gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ string, val any, _ time.Duration, _ ...cache.CallOption) error {
			raw, ok := val.([]byte)
			require.True(t, ok)
			written = raw
			return nil
		},
	)
	store.EXPECT().Get(gomock.Any(), "k").DoAndReturn(
		func(_ context.Context, _ string, _ ...cache.CallOption) (any, error) {
			return written, nil
		},
	)

	builder, err := NewBuilder[[]byte]("compress-bytes", store)
	require.NoError(t, err)
	c, err := builder.WithCompression(codec.GzipCompressionCodec{}).Build()
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "k", plain, time.Minute))
	require.NotEqual(t, plain, written)

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestBuilderCompressionWithCodec(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	store := mocks.NewMockStore(ctrl)

	type payload struct {
		Name string
		Age  int
	}
	input := payload{Name: "alice", Age: 18}
	var written []byte

	store.EXPECT().StoreName().Return("mock-store").Times(1)
	store.EXPECT().Set(gomock.Any(), "k", gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ string, val any, _ time.Duration, _ ...cache.CallOption) error {
			raw, ok := val.([]byte)
			require.True(t, ok)
			written = raw
			return nil
		},
	)
	store.EXPECT().Get(gomock.Any(), "k").DoAndReturn(
		func(_ context.Context, _ string, _ ...cache.CallOption) (any, error) {
			return written, nil
		},
	)

	builder, err := NewBuilder[payload]("compress-object", store)
	require.NoError(t, err)
	c, err := builder.
		WithCodec(codec.JSONCodec{}).
		WithCompression(codec.GzipCompressionCodec{}).
		Build()
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "k", input, time.Minute))
	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, input, got)
}

func TestBuilderLogicExpireWithCodecAndCompression(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	store := mocks.NewMockStore(ctrl)

	var written []byte
	store.EXPECT().StoreName().Return("mock-store").Times(1)
	store.EXPECT().Set(gomock.Any(), "k", gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ string, val any, _ time.Duration, _ ...cache.CallOption) error {
			raw, ok := val.([]byte)
			require.True(t, ok)
			written = raw
			return nil
		},
	)
	store.EXPECT().Get(gomock.Any(), "k").DoAndReturn(
		func(_ context.Context, _ string, _ ...cache.CallOption) (any, error) {
			return written, nil
		},
	)

	builder, err := NewBuilder[string]("logic-codec-compress", store)
	require.NoError(t, err)
	c, err := builder.
		WithLogicExpireEnabled(true).
		WithCodec(codec.JSONCodec{}).
		WithCompression(codec.GzipCompressionCodec{}).
		Build()
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))
	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, "v", got)
}

func TestBuilderLogicExpireWithBytesAndCompressionWithoutCodec(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	store := mocks.NewMockStore(ctrl)

	plain := []byte("logic-bytes")
	var written []byte
	store.EXPECT().StoreName().Return("mock-store").Times(1)
	store.EXPECT().Set(gomock.Any(), "k", gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ string, val any, _ time.Duration, _ ...cache.CallOption) error {
			raw, ok := val.([]byte)
			require.True(t, ok)
			written = raw
			return nil
		},
	)
	store.EXPECT().Get(gomock.Any(), "k").DoAndReturn(
		func(_ context.Context, _ string, _ ...cache.CallOption) (any, error) {
			return written, nil
		},
	)

	builder, err := NewBuilder[[]byte]("logic-bytes-compress", store)
	require.NoError(t, err)
	c, err := builder.
		WithLogicExpireEnabled(true).
		WithLogicExpireBytesAdapter(true).
		WithCompression(codec.GzipCompressionCodec{}).
		Build()
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "k", plain, time.Minute))
	require.NotEqual(t, plain, written)

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestBuilderLogicExpireWithBytesAndCompressionWithoutExplicitBytesAdapterShouldFail(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)

	builder, err := NewBuilder[[]byte]("logic-bytes-compress-no-explicit-adapter", store)
	require.NoError(t, err)
	_, err = builder.
		WithLogicExpireEnabled(true).
		WithCompression(codec.GzipCompressionCodec{}).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "try use builder.WithLogicExpireBytesAdapter(true) to enable it")
}

func TestBuilderCompressionWithoutAdapterShouldFail(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)

	builder, err := NewBuilder[string]("compress-no-adapter", store)
	require.NoError(t, err)
	_, err = builder.WithCompression(codec.GzipCompressionCodec{}).Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no adapter configured")
}

func TestBuilderCompressionWithCustomTypeAdapter(t *testing.T) {
	ctrl := gomock.NewController(t)

	ctx := context.Background()
	store := mocks.NewMockStore(ctrl)
	var written []byte

	store.EXPECT().StoreName().Return("mock-store").Times(1)
	store.EXPECT().Set(gomock.Any(), "k", gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ string, val any, _ time.Duration, _ ...cache.CallOption) error {
			raw, ok := val.([]byte)
			require.True(t, ok)
			written = raw
			return nil
		},
	)
	store.EXPECT().Get(gomock.Any(), "k").DoAndReturn(
		func(_ context.Context, _ string, _ ...cache.CallOption) (any, error) {
			return written, nil
		},
	)

	builder, err := NewBuilder[string]("compress-custom-adapter", store)
	require.NoError(t, err)
	c, err := builder.
		WithTypeAdapter(func(next cache.Cache[[]byte], _ *telemetry.Observable) (cache.Cache[string], error) {
			return &stringByteAdapter{Cache: next}, nil
		}).
		WithCompression(codec.GzipCompressionCodec{}).
		Build()
	require.NoError(t, err)

	require.NoError(t, c.Set(ctx, "k", "hello", time.Minute))
	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, "hello", got)
}

func TestBuilderWithFactoryConflictWithStagedFeatures(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockStore(ctrl)

	builder, err := NewBuilder[string]("factory-conflict", store)
	require.NoError(t, err)
	_, err = builder.
		WithFactory(func(store cache.Store, _ *telemetry.Observable) (cache.Cache[string], error) {
			return cache.NewBaseCache[string](store), nil
		}).
		WithCodec(codec.JSONCodec{}).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "WithFactory cannot be combined with staged features")
}

type stringByteAdapter struct {
	cache.Cache[[]byte]
}

func (s *stringByteAdapter) Get(ctx context.Context, key string, opts ...cache.CallOption) (string, error) {
	raw, err := s.Cache.Get(ctx, key, opts...)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *stringByteAdapter) Set(ctx context.Context, key string, val string, ttl time.Duration, opts ...cache.CallOption) error {
	return s.Cache.Set(ctx, key, []byte(val), ttl, opts...)
}

func (s *stringByteAdapter) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (string, time.Duration, error) {
	raw, ttl, err := s.Cache.GetWithTTL(ctx, key, opts...)
	if err != nil {
		return "", 0, err
	}
	return string(raw), ttl, nil
}

func (s *stringByteAdapter) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return s.Cache.Delete(ctx, key, opts...)
}

func (s *stringByteAdapter) Clear(ctx context.Context) error {
	return s.Cache.Clear(ctx)
}

var _ cache.Cache[string] = (*stringByteAdapter)(nil)
