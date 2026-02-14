package multicache

import (
	"context"
	"testing"
	"time"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

type mockMetrics struct {
	events []*telemetry.Event
}

func (m *mockMetrics) Record(ctx context.Context, e *telemetry.Event) error {
	m.events = append(m.events, e)
	return nil
}

type mockCache struct {
	val string
	err error
}

func (m *mockCache) Get(ctx context.Context, key string, opts ...cache.CallOption) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.val, nil
}

func (m *mockCache) Set(ctx context.Context, key string, val string, ttl time.Duration, opts ...cache.CallOption) error {
	return m.err
}

func (m *mockCache) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (string, time.Duration, error) {
	if m.err != nil {
		return "", 0, m.err
	}
	// Return a dummy TTL
	return m.val, time.Minute, nil
}

func (m *mockCache) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	return m.err
}

func (m *mockCache) Clear(ctx context.Context) error {
	return m.err
}

func (m *mockCache) Close() error { return nil }

func TestMultiCacheTelemetry(t *testing.T) {
	metrics := &mockMetrics{}
	ob := &telemetry.Observable{
		Metrics: metrics,
		Logger:  telemetry.SlogLogger(), // or mock logger if needed
	}

	c1 := &mockCache{val: "value1"}

	// Create MultiCache manually since we are in `package multicache` and can access `New`
	// But `New` takes `Config`.
	// Let's use `New` directly to test `MultiCache` logic, as `MultiBuilder` is in `cachalot` package (circular dependency if we use it here?)
	// `MultiBuilder` is in `d:\codes\cachalot\multi_cache.go` (package `cachalot`).
	// `MultiCache` is in `d:\codes\cachalot\core\multicache\multi_cache.go` (package `multicache`).
	// We are testing `core/multicache`.

	cfg := Config[string]{
		Observable: ob,
		// minimal config
		FetchPolicy: func(ctx context.Context, ctx2 *FetchContext[string]) (string, []FailedCache[string], error) {
			val, err := ctx2.Cache.Caches()[0].Get(ctx, ctx2.Key)
			if err != nil {
				return "", []FailedCache[string]{{Cache: ctx2.Cache.Caches()[0], Err: err}}, nil
			}
			return val, nil, nil
		},
		WriteBackCacheFilter: func(ctx context.Context, ctx2 *FetchContext[string], failed []FailedCache[string]) []cache.Cache[string] {
			return nil
		},
		WriteBackFn: func(ctx context.Context, ctx2 *FetchContext[string], caches []cache.Cache[string]) error {
			return nil
		},
	}

	mc, err := New("cache", cfg, c1)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()

	// Test Get
	_, _ = mc.Get(ctx, "key")
	if len(metrics.events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(metrics.events))
	} else {
		if metrics.events[0].Op != telemetry.OpGet {
			t.Errorf("Expected OpGet, got %v", metrics.events[0].Op)
		}
	}

	// Test Set
	_ = mc.Set(ctx, "key", "val", time.Minute)
	if len(metrics.events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(metrics.events))
	} else {
		if metrics.events[1].Op != telemetry.OpSet {
			t.Errorf("Expected OpSet, got %v", metrics.events[1].Op)
		}
	}

	// Test Delete
	_ = mc.Delete(ctx, "key")
	if len(metrics.events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(metrics.events))
	} else {
		if metrics.events[2].Op != telemetry.OpDelete {
			t.Errorf("Expected OpDelete, got %v", metrics.events[2].Op)
		}
	}

	// Test Clear
	_ = mc.Clear(ctx)
	if len(metrics.events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(metrics.events))
	} else {
		if metrics.events[3].Op != telemetry.OpClear {
			t.Errorf("Expected OpClear, got %v", metrics.events[3].Op)
		}
	}
}
