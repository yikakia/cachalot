package ristretto

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/stores/storetests"
)

// newTestStore 创建测试用的 Store 实例
func newTestStore(t *testing.T) *Store {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7,     // 计数器数量
		MaxCost:     1 << 30, // 最大 cost
		BufferItems: 64,      // 缓冲区大小
	})
	require.NoError(t, err)
	return New(cache, WithStoreName("test-ristretto"))
}

func TestStoreSuites(t *testing.T) {
	storetests.RunStoreTestSuites(t,
		func(t *testing.T) cache.Store {
			return newTestStore(t)
		},
		storetests.WithSetOptions(WithSynchronousSet(true)),
	)
}

// ==================== StoreName Tests ====================

func TestStoreName_NotEmpty(t *testing.T) {
	s := newTestStore(t)

	name := s.StoreName()
	assert.NotEmpty(t, name)
	assert.Equal(t, "test-ristretto", name)
}

func TestStoreName_Consistent(t *testing.T) {
	s := newTestStore(t)

	// 多次调用应返回相同的名称
	name1 := s.StoreName()
	name2 := s.StoreName()
	name3 := s.StoreName()

	assert.Equal(t, name1, name2)
	assert.Equal(t, name2, name3)
}

func TestStoreName_DefaultName(t *testing.T) {
	// 不使用 WithStoreName，应该使用默认名称
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	require.NoError(t, err)

	s := New(cache) // 不传 WithStoreName
	assert.Equal(t, "ristretto", s.StoreName())
}

// ==================== Additional Tests ====================

func TestStore_ImplementsInterface(t *testing.T) {
	// 验证 Store 实现了 cache.Store 接口
	var _ cache.Store = (*Store)(nil)
}

// ==================== DifferentValueTypes Tests ====================

func TestStore_DifferentValueTypes(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	testCases := []struct {
		name  string
		key   string
		value any
	}{
		{"string", "string-key", "string-value"},
		{"int", "int-key", 42},
		{"float", "float-key", 3.14},
		{"bool", "bool-key", true},
		{"slice", "slice-key", []int{1, 2, 3}},
		{"map", "map-key", map[string]int{"a": 1, "b": 2}},
		{"struct", "struct-key", struct{ Name string }{"test"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Set(ctx, tc.key, tc.value, time.Minute, WithSynchronousSet(true))
			require.NoError(t, err)

			val, err := s.Get(ctx, tc.key)
			assert.NoError(t, err)
			assert.Equal(t, tc.value, val)
		})
	}
}
