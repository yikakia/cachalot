package storetests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yikakia/cachalot/core/cache"
)

func RunStoreTestSuites(t *testing.T, newStore func(*testing.T) cache.Store, opts ...Option) {
	config := NewConfig()
	for _, opt := range opts {
		opt.Apply(config)
	}

	skipSet := map[string]bool{}
	for _, test := range config.SkipTests {
		skipSet[test] = true
	}

	trySkip := func(t *testing.T) {
		if skipSet[t.Name()] {
			t.Skip()
		}
	}

	waitForCache := func(t *testing.T, s cache.Store) {
		config.WaitingAfterWrite(t, s)
	}
	encodeSetValue := func(v string) any {
		return config.EncodeSetValue(v)
	}
	assertValue := func(t *testing.T, got any, expected string) {
		config.AssertValue(t, got, expected)
	}

	t.Run("Get", func(t *testing.T) {
		trySkip(t)
		t.Run("ExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 先设置值
			err := s.Set(ctx, "key1", encodeSetValue("value1"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 获取已存在的 key
			val, err := s.Get(ctx, "key1")
			assert.NoError(t, err)
			assertValue(t, val, "value1")
		})

		t.Run("NonExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 获取不存在的 key
			val, err := s.Get(ctx, "non-existing-key")
			assert.Nil(t, val)
			assert.Error(t, err)
			assert.ErrorIs(t, err, cache.ErrNotFound)
		})

		t.Run("ExpiredKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置一个很短 TTL 的值
			// 存在一些实现 只支持秒级的精度
			err := s.Set(ctx, "expired-key", encodeSetValue("value"), time.Second, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 等待过期
			time.Sleep(time.Millisecond * 1100)

			// 获取已过期的 key
			val, err := s.Get(ctx, "expired-key")
			assert.Nil(t, val)
			assert.Error(t, err)
			assert.ErrorIs(t, err, cache.ErrNotFound)
		})

		t.Run("WithOptions", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 先设置值
			err := s.Set(ctx, "option-key", encodeSetValue("option-value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 使用自定义 GetOption
			customOpt := cache.WithOptionCustomField("custom", "field")
			val, err := s.Get(ctx, "option-key", customOpt)
			assert.NoError(t, err)
			assertValue(t, val, "option-value")
		})

		t.Run("ContextCancelled", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)

			// 创建已取消的 context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// 先设置值
			err := s.Set(context.Background(), "cancel-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 注意: 具体实现可能不检查 context
			val, err := s.Get(ctx, "cancel-key")
			assert.Nil(t, val)
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)
		})
	})

	t.Run("Set", func(t *testing.T) {
		trySkip(t)
		t.Run("NewKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置新 key
			err := s.Set(ctx, "new-key", encodeSetValue("new-value"), time.Minute, config.SetOptions...)
			assert.NoError(t, err)
			waitForCache(t, s)

			// 验证能获取到
			val, err := s.Get(ctx, "new-key")
			assert.NoError(t, err)
			assertValue(t, val, "new-value")
		})

		t.Run("OverwriteExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置初始值
			err := s.Set(ctx, "overwrite-key", encodeSetValue("old-value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 覆盖值
			err = s.Set(ctx, "overwrite-key", encodeSetValue("new-value"), time.Minute, config.SetOptions...)
			assert.NoError(t, err)
			waitForCache(t, s)

			// 验证新值
			val, err := s.Get(ctx, "overwrite-key")
			assert.NoError(t, err)
			assertValue(t, val, "new-value")
		})

		t.Run("ZeroTTL", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// TTL 为 0 表示永不过期
			err := s.Set(ctx, "zero-ttl-key", encodeSetValue("value"), 0, config.SetOptions...)
			assert.NoError(t, err)
			waitForCache(t, s)

			// 等待一段时间后仍能获取
			time.Sleep(50 * time.Millisecond)
			val, err := s.Get(ctx, "zero-ttl-key")
			assert.NoError(t, err)
			assertValue(t, val, "value")
		})

		t.Run("NegativeTTL", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 负数 TTL 视为无效
			err := s.Set(ctx, "negative-ttl-key", encodeSetValue("value"), -1*time.Second, config.SetOptions...)
			assert.ErrorIs(t, err, cache.ErrInvalidTTL)
		})

		t.Run("WithOptions", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 使用自定义 SetOption (Generic cache option) alongside config options
			opts := append([]cache.CallOption{}, config.SetOptions...)
			opts = append(opts, cache.WithOptionCustomField("cost", int64(10)))

			err := s.Set(ctx, "options-key", encodeSetValue("value"), time.Minute, opts...)
			assert.NoError(t, err)
			waitForCache(t, s)

			// 验证值已设置
			val, err := s.Get(ctx, "options-key")
			assert.NoError(t, err)
			assertValue(t, val, "value")
		})

		t.Run("ContextCancelled", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)

			// 创建已取消的 context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := s.Set(ctx, "cancel-set-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)
		})
	})

	t.Run("GetWithTTL", func(t *testing.T) {
		trySkip(t)
		t.Run("ExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置值
			err := s.Set(ctx, "ttl-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 获取值和 TTL
			val, ttl, err := s.GetWithTTL(ctx, "ttl-key")
			assert.NoError(t, err)
			assertValue(t, val, "value")
			// TTL 应该在合理范围内
			assert.True(t, ttl > 0 && ttl <= time.Minute, "TTL should be positive and <= 1 minute, got: %v", ttl)
		})

		t.Run("NonExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 获取不存在的 key
			val, ttl, err := s.GetWithTTL(ctx, "non-existing-ttl-key")
			assert.Nil(t, val)
			assert.Zero(t, ttl)
			assert.Error(t, err)
			assert.ErrorIs(t, err, cache.ErrNotFound)
		})

		t.Run("TTLDecreasing", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置值
			err := s.Set(ctx, "decreasing-ttl-key", encodeSetValue("value"), 5*time.Second, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 第一次获取 TTL
			_, ttl1, err := s.GetWithTTL(ctx, "decreasing-ttl-key")
			require.NoError(t, err)

			// 等待一段时间
			time.Sleep(100 * time.Millisecond)

			// 第二次获取 TTL
			_, ttl2, err := s.GetWithTTL(ctx, "decreasing-ttl-key")
			require.NoError(t, err)

			// TTL 应该减少
			assert.True(t, ttl2 < ttl1, "TTL should decrease: first=%v, second=%v", ttl1, ttl2)
		})

		t.Run("NoExpiry", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置永不过期的值 (TTL = 0)
			err := s.Set(ctx, "no-expiry-key", encodeSetValue("value"), 0, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 获取 TTL
			val, ttl, err := s.GetWithTTL(ctx, "no-expiry-key")
			assert.NoError(t, err)
			assertValue(t, val, "value")
			// 永不过期的 key，TTL 应该返回 0
			assert.Zero(t, ttl, "TTL for no-expiry key should be 0")
		})

		t.Run("WithOptions", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置值
			err := s.Set(ctx, "ttl-option-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 使用自定义 GetOption
			customOpt := cache.WithOptionCustomField("custom", "field")
			val, ttl, err := s.GetWithTTL(ctx, "ttl-option-key", customOpt)
			assert.NoError(t, err)
			assertValue(t, val, "value")
			assert.True(t, ttl > 0, "TTL should be positive")
		})
	})

	t.Run("Delete", func(t *testing.T) {
		trySkip(t)
		t.Run("ExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 先设置值
			err := s.Set(ctx, "delete-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 删除
			err = s.Delete(ctx, "delete-key")
			assert.NoError(t, err)

			// 验证已删除
			val, err := s.Get(ctx, "delete-key")
			assert.Nil(t, val)
			assert.Error(t, err)
			assert.ErrorIs(t, err, cache.ErrNotFound)
		})

		t.Run("NonExistingKey", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 删除不存在的 key 应该不报错（幂等操作）
			err := s.Delete(ctx, "non-existing-delete-key")
			assert.NoError(t, err)
		})

		t.Run("WithOptions", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 先设置值
			err := s.Set(ctx, "delete-option-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 使用 DeleteOption 删除
			err = s.Delete(ctx, "delete-option-key")
			assert.NoError(t, err)

			// 验证已删除
			_, err = s.Get(ctx, "delete-option-key")
			assert.True(t, errors.Is(err, cache.ErrNotFound))
		})

		t.Run("ContextCancelled", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)

			// 先设置值
			err := s.Set(context.Background(), "cancel-delete-key", encodeSetValue("value"), time.Minute, config.SetOptions...)
			require.NoError(t, err)
			waitForCache(t, s)

			// 创建已取消的 context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = s.Delete(ctx, "cancel-delete-key")
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)
		})
	})

	t.Run("Clear", func(t *testing.T) {
		trySkip(t)
		t.Run("EmptyStore", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 空 store 调用 Clear 不应报错
			err := s.Clear(ctx)
			assert.NoError(t, err)
		})

		t.Run("NonEmptyStore", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 设置多个值
			keys := []string{"clear-key1", "clear-key2", "clear-key3"}
			for _, key := range keys {
				err := s.Set(ctx, key, encodeSetValue("value"), time.Minute, config.SetOptions...)
				require.NoError(t, err)
			}
			waitForCache(t, s)

			// 验证值存在
			for _, key := range keys {
				_, err := s.Get(ctx, key)
				require.NoError(t, err)
			}

			// 清空
			err := s.Clear(ctx)
			assert.NoError(t, err)

			// 验证所有 key 都已删除
			for _, key := range keys {
				val, err := s.Get(ctx, key)
				assert.Nil(t, val)
				assert.True(t, errors.Is(err, cache.ErrNotFound), "key %s should be deleted", key)
			}
		})

		t.Run("ContextCancelled", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)

			// 创建已取消的 context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := s.Clear(ctx)
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)
		})

		t.Run("MultipleTimes", func(t *testing.T) {
			trySkip(t)
			s := newStore(t)
			ctx := context.Background()

			// 多次调用 Clear 不应报错
			for i := 0; i < 3; i++ {
				err := s.Clear(ctx)
				assert.NoError(t, err)
			}
		})
	})

}
