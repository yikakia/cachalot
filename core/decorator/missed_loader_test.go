package decorator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/decorator"
	"github.com/yikakia/cachalot/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestMissedLoaderDecorator_Get(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	val := "test-val"
	ttl := time.Minute

	t.Run("cache hit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return(val, nil)

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:  mockCache,
			LoadFn: nil,
		})

		res, err := d.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, val, res)
	})

	t.Run("cache miss with loader", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return("", cache.ErrNotFound)
		mockCache.EXPECT().Set(gomock.Any(), gomock.Eq(key), gomock.Eq(val), gomock.Eq(ttl)).Return(nil)

		loader := func(ctx context.Context, k string, _ ...cache.CallOption) (string, error) {
			assert.Equal(t, key, k)
			return val, nil
		}

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:        mockCache,
			LoadFn:       loader,
			WriteBackTTL: ttl,
		})

		res, err := d.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, val, res)
	})

	t.Run("cache miss with loader error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return("", cache.ErrNotFound)

		loaderErr := errors.New("loader error")
		loader := func(ctx context.Context, k string, _ ...cache.CallOption) (string, error) {
			return "", loaderErr
		}

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:  mockCache,
			LoadFn: loader,
		})

		res, err := d.Get(ctx, key)
		assert.ErrorIs(t, err, loaderErr)
		assert.Empty(t, res)
	})

	t.Run("cache miss without loader", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCache := mocks.NewMockCache[string](ctrl)
		mockCache.EXPECT().Get(gomock.Any(), gomock.Eq(key)).Return("", cache.ErrNotFound)

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:  mockCache,
			LoadFn: nil,
		})

		res, err := d.Get(ctx, key)
		assert.ErrorIs(t, err, cache.ErrNotFound)
		assert.Empty(t, res)
	})
}
