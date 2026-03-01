package freecache

import (
	"testing"

	"github.com/coocood/freecache"
	"github.com/stretchr/testify/assert"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/stores/storetests"
)

// newTestStore 创建测试用的 Store 实例
func newTestStore(t *testing.T) *Store {
	// 100MB
	client := freecache.NewCache(100 * 1024 * 1024)
	return New(client, WithStoreName("test-freecache"))
}

func TestFreeCache(t *testing.T) {
	storetests.RunStoreTestSuites(t,
		func(t *testing.T) cache.Store {
			return newTestStore(t)
		},
		storetests.WithEncodeSetValue(func(v string) any {
			return []byte(v)
		}),
		storetests.WithAssertValue(func(t *testing.T, got any, expected string) {
			assert.Equal(t, []byte(expected), got)
		}),
	)
}

// ==================== StoreName Tests ====================

func TestStoreName_NotEmpty(t *testing.T) {
	s := newTestStore(t)

	name := s.StoreName()
	assert.NotEmpty(t, name)
	assert.Equal(t, "test-freecache", name)
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
	client := freecache.NewCache(100 * 1024 * 1024)
	s := New(client) // 不传 WithStoreName
	assert.Equal(t, "freecache", s.StoreName())
}

// ==================== Additional Tests ====================

func TestStore_ImplementsInterface(t *testing.T) {
	// 验证 Store 实现了 cache.Store 接口
	var _ cache.Store = (*Store)(nil)
}
