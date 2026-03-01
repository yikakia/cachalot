package integration

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/valkey-io/valkey-go"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/stores/storetests"
	store_valkey "github.com/yikakia/cachalot/stores/valkey"
)

func newValkeyClient(t *testing.T) valkey.Client {
	ctx := context.Background()

	opts := []testcontainers.ContainerCustomizer{
		//testcontainers.WithLogger(tclog.TestLogger(t)),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("* Ready to accept connections"),
				wait.ForExposedPort(),
			),
		),
	}

	valkeyContainer, err := tcvalkey.Run(ctx, "valkey/valkey:latest", opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, valkeyContainer.Terminate(context.TODO()))
	})

	endpoint, err := valkeyContainer.ConnectionString(context.TODO())
	require.NoError(t, err)

	opt, err := valkey.ParseURL(endpoint)
	require.NoError(t, err)
	client, err := valkey.NewClient(opt)
	require.NoError(t, err)

	return client
}

func newValkeyStore(t *testing.T) cache.Store {
	return store_valkey.New(newValkeyClient(t),
		store_valkey.WithClientSideCacheExpiration(time.Second))
}

func TestValkey(t *testing.T) {
	storetests.RunStoreTestSuites(t, newValkeyStore,
		storetests.WithSkipTests("TestValkey/Clear/NonEmptyStore"),
		storetests.WithEncodeSetValue(func(v string) any {
			return []byte(v)
		}),
		storetests.WithAssertValue(func(t *testing.T, got any, expected string) {
			raw, ok := got.([]byte)
			if !ok {
				t.Fatalf("want []byte got %T", got)
			}
			assert.True(t, bytes.Equal(raw, []byte(expected)))
		}))
}

//func TestFlush(t *testing.T) {
//	for range 100 {
//		t.Run(strconv.Itoa(100), func(t *testing.T) {
//			vc := newValkeyClient(t)
//
//			keys := []string{"key1", "key2", "key3"}
//			for _, key := range keys {
//				cmd := vc.B().Set().Key(key).Value(key + "value").Build()
//				err := vc.Do(context.Background(), cmd).Error()
//				assert.NoError(t, err)
//			}
//
//			for _, key := range keys {
//				cmd := vc.B().Get().Key(key).Build()
//				err := vc.Do(context.Background(), cmd).Error()
//				assert.NoError(t, err)
//			}
//
//			cmd := vc.B().Flushall().Sync().Build()
//			require.NoError(t, vc.Do(context.Background(), cmd).Error())
//
//			for _, key := range keys {
//				cmd := vc.B().Get().Key(key).Build()
//				err := vc.Do(context.Background(), cmd).Error()
//				assert.ErrorIs(t, err, valkey.Nil, "key %s should be deleted", key)
//			}
//		})
//	}
//}
