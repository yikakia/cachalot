package integration

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/yikakia/cachalot/core/cache"
	store_redis "github.com/yikakia/cachalot/stores/redis"
	"github.com/yikakia/cachalot/stores/storetests"
)

func newRedisClient(t *testing.T) *redis.Client {
	ctx := context.Background()
	redisC, err := testcontainers.Run(
		ctx, "redis:latest",
		testcontainers.WithExposedPorts("6379/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("6379/tcp"),
			wait.ForLog("Ready to accept connections")),
	)
	testcontainers.CleanupContainer(t, redisC)
	require.NoError(t, err)

	endpoint, err := redisC.Endpoint(ctx, "")
	require.NoError(t, err)
	return redis.NewClient(&redis.Options{Addr: endpoint})
}
func newStore(t *testing.T) cache.Store {
	c := sync.OnceValue(func() *redis.Client {
		return newRedisClient(t)
	})

	require.NoError(t, c().FlushDB(context.TODO()).Err())

	return store_redis.New(c(), store_redis.WithStoreName("test_redis"))

}

func TestRedis(t *testing.T) {
	storetests.RunStoreTestSuites(t, newStore,
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
