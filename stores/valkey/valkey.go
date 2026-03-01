package valkey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
	"github.com/yikakia/cachalot/core/cache"
)

const defaultClientSideCacheExpiration = time.Minute

func New(client valkey.Client, opts ...Option) *Store {
	s := &Store{
		client:                    client,
		name:                      "valkey",
		clientSideCacheExpiration: defaultClientSideCacheExpiration,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type Store struct {
	name                      string
	client                    valkey.Client
	clientSideCacheExpiration time.Duration
}

func (s *Store) Get(ctx context.Context, key string, opts ...cache.CallOption) (any, error) {
	cmd := s.client.B().Get().Key(key).Cache()
	bytes, err := s.client.DoCache(ctx, cmd, s.clientSideCacheExpiration).AsBytes()
	if valkey.IsValkeyNil(err) {
		return nil, cache.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (s *Store) Set(ctx context.Context, key string, val any, ttl time.Duration, opts ...cache.CallOption) error {
	if ttl < 0 {
		return cache.ErrInvalidTTL
	}

	byteVal, ok := val.([]byte)
	if !ok {
		return fmt.Errorf("valkey.Set expects byte array: %s", cache.ErrTypeMissMatch)
	}
	var cmd valkey.Completed
	cmdBuilder := s.client.B().Set().Key(key).Value(valkey.BinaryString(byteVal))
	if ttl > 0 {
		// valkey 不接受 ttl == 0
		cmdBuilder.Px(ttl)
	}
	cmd = cmdBuilder.Build()
	err := s.client.Do(ctx, cmd).Error()
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) GetWithTTL(ctx context.Context, key string, opts ...cache.CallOption) (any, time.Duration, error) {
	getCMD := s.client.B().Get().Key(key).Build()
	ttlCMD := s.client.B().Pttl().Key(key).Build()
	rets := s.client.DoMulti(ctx, getCMD, ttlCMD)

	if len(rets) != 2 {
		return nil, 0, fmt.Errorf("unknown err: valkey.GetWithTTL expects 2 results, but got %d", len(rets))
	}

	var joinedErr error

	val, err := rets[0].AsBytes()
	if valkey.IsValkeyNil(err) {
		return nil, 0, fmt.Errorf("key:%s is nil: %w", key, cache.ErrNotFound)
	}
	joinedErr = errors.Join(joinedErr, err)
	pttl, err := rets[1].ToInt64()
	joinedErr = errors.Join(joinedErr, err)

	if joinedErr != nil {
		return nil, 0, fmt.Errorf("valkey.GetWithTTL parse result failed: %w", joinedErr)
	}

	// TODO -2 ?
	if pttl == -1 {
		pttl = 0
	}

	return val, time.Duration(pttl) * time.Millisecond, nil
}

func (s *Store) Delete(ctx context.Context, key string, opts ...cache.CallOption) error {
	cmd := s.client.B().Del().Key(key).Build()
	return s.client.Do(ctx, cmd).Error()
}

func (s *Store) Clear(ctx context.Context) error {
	cmd := s.client.B().Flushall().Sync().Build()
	return s.client.Do(ctx, cmd).Error()
}

func (s *Store) StoreName() string {
	return s.name
}

var _ cache.Store = (*Store)(nil)
