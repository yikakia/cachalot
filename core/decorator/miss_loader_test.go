package decorator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/decorator"
)

// MockCache implements cache.Cache[T]
type MockCache[T any] struct {
	mock.Mock
}

func (m *MockCache[T]) Get(ctx context.Context, key string, opts ...cache.CallOption) (T, error) {
	args := m.Called(ctx, key, opts)
	return args.Get(0).(T), args.Error(1)
}

func (m *MockCache[T]) Set(ctx context.Context, key string, val T, ttl time.Duration, opts ...cache.CallOption) error {
	args := m.Called(ctx, key, val, ttl, opts)
	return args.Error(0)
}

func (m *MockCache[T]) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (T, time.Duration, error) {
	args := m.Called(ctx, key, opts)
	return args.Get(0).(T), args.Get(1).(time.Duration), args.Error(2)
}

func (m *MockCache[T]) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	args := m.Called(ctx, key, opts)
	return args.Error(0)
}

func (m *MockCache[T]) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestLoaderDecorator_Get(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	val := "test-val"
	ttl := time.Minute

	t.Run("cache hit", func(t *testing.T) {
		mockCache := new(MockCache[string])
		mockCache.On("Get", ctx, key, mock.Anything).Return(val, nil)

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:  mockCache,
			LoadFn: nil,
		})

		res, err := d.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, val, res)
		mockCache.AssertExpectations(t)
	})

	t.Run("cache miss with loader", func(t *testing.T) {
		mockCache := new(MockCache[string])
		mockCache.On("Get", ctx, key, mock.Anything).Return("", cache.ErrNotFound)
		mockCache.On("Set", ctx, key, val, ttl, mock.Anything).Return(nil)

		loader := func(ctx context.Context, k string) (string, error) {
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
		mockCache.AssertExpectations(t)
	})

	t.Run("cache miss with loader error", func(t *testing.T) {
		mockCache := new(MockCache[string])
		mockCache.On("Get", ctx, key, mock.Anything).Return("", cache.ErrNotFound)

		loaderErr := errors.New("loader error")
		loader := func(ctx context.Context, k string) (string, error) {
			return "", loaderErr
		}

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:  mockCache,
			LoadFn: loader,
		})

		res, err := d.Get(ctx, key)
		assert.ErrorIs(t, err, loaderErr)
		assert.Empty(t, res)
		mockCache.AssertExpectations(t)
	})

	t.Run("cache miss without loader", func(t *testing.T) {
		mockCache := new(MockCache[string])
		mockCache.On("Get", ctx, key, mock.Anything).Return("", cache.ErrNotFound)

		d := decorator.NewMissedLoaderDecorator(decorator.MissedLoaderDecoratorConfig[string]{
			Cache:  mockCache,
			LoadFn: nil,
		})

		res, err := d.Get(ctx, key)
		assert.ErrorIs(t, err, cache.ErrNotFound)
		assert.Empty(t, res)
		mockCache.AssertExpectations(t)
	})
}
