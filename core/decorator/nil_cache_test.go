package decorator_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/decorator"
	"github.com/yikakia/cachalot/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestNilCacheDecorator_Get(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	val := "test-val"
	ttl := time.Minute

	t.Run("cache hit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return(val, nil)
		d := decorator.NewNilCacheDecorator(decorator.NilCacheConfig[string]{
			Cache: mockCache,
		})

		get, err := d.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, val, get)
	})

	t.Run("cache miss with protection", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return("", cache.ErrNotFound)
		// when miss will call set
		mockCache.EXPECT().Set(gomock.Any(), gomock.Eq(key), gomock.Eq(val), gomock.Eq(ttl)).Return(nil)

		d := decorator.NewNilCacheDecorator(decorator.NilCacheConfig[string]{
			Cache: mockCache,
			ProtectionFn: func(_key string) string {
				require.Equal(t, key, _key)
				return val
			},
			WriteBackTTL: ttl,
		})

		get, err := d.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, val, get)
	})

	t.Run("cache miss without protection", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return("", cache.ErrNotFound)

		d := decorator.NewNilCacheDecorator(decorator.NilCacheConfig[string]{
			Cache: mockCache,
		})
		_, err := d.Get(ctx, key)
		assert.ErrorIs(t, err, cache.ErrNotFound)
	})
}
