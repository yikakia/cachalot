package cachalot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yikakia/cachalot/core/cache/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewBuilderValidation(t *testing.T) {
	_, err := NewBuilder[string]("", nil)
	require.Error(t, err)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	store := mocks.NewMockStore(ctrl)

	_, err = NewBuilder[string]("cache", store)
	require.NoError(t, err)
}

func TestBuilderBuildAndDelegateToStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

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
	t.Cleanup(ctrl.Finish)

	store := mocks.NewMockStore(ctrl)
	builder, err := NewBuilder[string]("single-cache", store)
	require.NoError(t, err)

	_, err = builder.WithLogicExpireDefaultLogicTTL(-time.Second).Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "logicExpireDefaultLogicTTL")
}
