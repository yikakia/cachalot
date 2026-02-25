package multicache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
	"github.com/yikakia/cachalot/internal/mocks"
	"go.uber.org/mock/gomock"
)

type mockMetrics struct {
	events []*telemetry.Event
}

func (m *mockMetrics) Record(ctx context.Context, e *telemetry.Event) error {
	m.events = append(m.events, e)
	return nil
}

func TestMultiCacheTelemetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	ctx := context.Background()
	metrics := &mockMetrics{}
	ob := &telemetry.Observable{Metrics: metrics, Logger: telemetry.SlogLogger()}

	c1 := mocks.NewMockCache[string](ctrl)
	c1.EXPECT().Get(gomock.Any(), "key").Return("value1", nil)
	c1.EXPECT().Set(gomock.Any(), "key", "val", time.Minute).Return(nil)
	c1.EXPECT().Delete(gomock.Any(), "key").Return(nil)
	c1.EXPECT().Clear(gomock.Any()).Return(nil)

	cfg := Config[string]{
		Observable: ob,
		FetchPolicy: func(ctx context.Context, ctx2 *FetchContext[string]) (string, []FailedCache[string], error) {
			val, err := ctx2.MultiCache.Caches()[0].Get(ctx, ctx2.Key)
			if err != nil {
				return "", []FailedCache[string]{{Cache: ctx2.MultiCache.Caches()[0], Err: err}}, nil
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
	require.NoError(t, err)

	_, _ = mc.Get(ctx, "key")
	require.Len(t, metrics.events, 1)
	require.Equal(t, telemetry.OpGet, metrics.events[0].Op)

	require.NoError(t, mc.Set(ctx, "key", "val", time.Minute))
	require.Len(t, metrics.events, 2)
	require.Equal(t, telemetry.OpSet, metrics.events[1].Op)

	require.NoError(t, mc.Delete(ctx, "key"))
	require.Len(t, metrics.events, 3)
	require.Equal(t, telemetry.OpDelete, metrics.events[2].Op)

	require.NoError(t, mc.Clear(ctx))
	require.Len(t, metrics.events, 4)
	require.Equal(t, telemetry.OpClear, metrics.events[3].Op)
}

func TestMultiCacheGetWriteBackErrorHandling(t *testing.T) {
	ctx := context.Background()
	writeBackErr := errors.New("write back failed")

	newConfig := func(mode ErrorHandleMode) Config[string] {
		return Config[string]{
			Observable: &telemetry.Observable{Metrics: telemetry.NoopMetrics(), Logger: telemetry.SlogLogger()},
			FetchPolicy: func(ctx context.Context, ctx2 *FetchContext[string]) (string, []FailedCache[string], error) {
				return "from-l2", []FailedCache[string]{{Err: errors.New("l1 miss")}}, nil
			},
			WriteBackCacheFilter: func(ctx context.Context, ctx2 *FetchContext[string], failed []FailedCache[string]) []cache.Cache[string] {
				return nil
			},
			WriteBackFn: func(ctx context.Context, ctx2 *FetchContext[string], caches []cache.Cache[string]) error {
				return writeBackErr
			},
			ErrorHandleMode: mode,
		}
	}

	t.Run("tolerant mode returns value and hides write back error", func(t *testing.T) {
		mc, err := New("cache", newConfig(ErrorHandleTolerant))
		require.NoError(t, err)

		v, err := mc.Get(ctx, "k")
		require.NoError(t, err)
		require.Equal(t, "from-l2", v)
	})

	t.Run("strict mode returns write back error", func(t *testing.T) {
		mc, err := New("cache", newConfig(ErrorHandleStrict))
		require.NoError(t, err)

		v, err := mc.Get(ctx, "k")
		require.ErrorIs(t, err, writeBackErr)
		require.Empty(t, v)
	})
}
