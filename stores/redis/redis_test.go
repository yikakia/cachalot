package redis

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/stores/storetests"
)

type fakeEntry struct {
	value  any
	expiry time.Time
}

type fakeClient struct {
	mu   sync.RWMutex
	data map[string]fakeEntry
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		data: map[string]fakeEntry{},
	}
}

func (f *fakeClient) Get(ctx context.Context, key string) *goredis.StringCmd {
	if err := ctx.Err(); err != nil {
		return goredis.NewStringResult("", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	entry, ok := f.data[key]
	if !ok {
		return goredis.NewStringResult("", goredis.Nil)
	}
	if !entry.expiry.IsZero() && time.Now().After(entry.expiry) {
		delete(f.data, key)
		return goredis.NewStringResult("", goredis.Nil)
	}
	raw, ok := entry.value.([]byte)
	if !ok {
		return goredis.NewStringResult("", goredis.Nil)
	}
	return goredis.NewStringResult(string(raw), nil)
}

func (f *fakeClient) Set(ctx context.Context, key string, value any, expiration time.Duration) *goredis.StatusCmd {
	if err := ctx.Err(); err != nil {
		return goredis.NewStatusResult("", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	entry := fakeEntry{value: value}
	if expiration > 0 {
		entry.expiry = time.Now().Add(expiration)
	}
	f.data[key] = entry
	return goredis.NewStatusResult("OK", nil)
}

func (f *fakeClient) TTL(ctx context.Context, key string) *goredis.DurationCmd {
	if err := ctx.Err(); err != nil {
		return goredis.NewDurationResult(0, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	entry, ok := f.data[key]
	if !ok {
		return goredis.NewDurationResult(-2*time.Second, nil)
	}
	if !entry.expiry.IsZero() && time.Now().After(entry.expiry) {
		delete(f.data, key)
		return goredis.NewDurationResult(-2*time.Second, nil)
	}
	if entry.expiry.IsZero() {
		return goredis.NewDurationResult(-1*time.Second, nil)
	}
	return goredis.NewDurationResult(time.Until(entry.expiry), nil)
}

func (f *fakeClient) Del(ctx context.Context, keys ...string) *goredis.IntCmd {
	if err := ctx.Err(); err != nil {
		return goredis.NewIntResult(0, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if _, ok := f.data[key]; ok {
			delete(f.data, key)
			deleted++
		}
	}
	return goredis.NewIntResult(deleted, nil)
}

func (f *fakeClient) FlushDB(ctx context.Context) *goredis.StatusCmd {
	if err := ctx.Err(); err != nil {
		return goredis.NewStatusResult("", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = map[string]fakeEntry{}
	return goredis.NewStatusResult("OK", nil)
}

func newTestStore() *Store {
	return New(newFakeClient(), WithStoreName("test-redis"))
}

func TestStoreSuites(t *testing.T) {
	storetests.RunStoreTestSuites(t, func(t *testing.T) cache.Store {
		return newTestStore()
	},
		storetests.WithEncodeSetValue(func(v string) any {
			return []byte(v)
		}),
		storetests.WithAssertValue(func(t *testing.T, got any, expected string) {
			raw, ok := got.([]byte)
			if !ok {
				t.Fatalf("want []byte got %T", got)
			}
			assert.True(t, bytes.Equal(raw, []byte(expected)))
		}),
	)
}

func TestStoreName_DefaultName(t *testing.T) {
	s := New(newFakeClient())
	assert.Equal(t, "redis", s.StoreName())
}

func TestStoreName_CustomName(t *testing.T) {
	s := newTestStore()
	assert.Equal(t, "test-redis", s.StoreName())
}

func TestSet_RejectNonBytes(t *testing.T) {
	s := newTestStore()
	err := s.Set(context.Background(), "invalid-type", "value", time.Minute)
	assert.True(t, errors.Is(err, cache.ErrTypeMissMatch))
}
