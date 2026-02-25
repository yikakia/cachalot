package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	cachecore "github.com/yikakia/cachalot/core/cache"
)

type mapItem struct {
	value    any
	expireAt time.Time
}

type SlowMapStore struct {
	name      string
	readDelay time.Duration
	mu        sync.RWMutex
	data      map[string]mapItem
}

func NewSlowMapStore(name string, readDelay time.Duration) *SlowMapStore {
	return &SlowMapStore{name: name, readDelay: readDelay, data: make(map[string]mapItem)}
}

func (s *SlowMapStore) Get(ctx context.Context, key string, opts ...cachecore.CallOption) (any, error) {
	_ = ctx
	_ = opts
	time.Sleep(s.readDelay)

	s.mu.RLock()
	item, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("key=%s not found: %w", key, cachecore.ErrNotFound)
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return nil, fmt.Errorf("key=%s expired: %w", key, cachecore.ErrNotFound)
	}
	return item.value, nil
}

func (s *SlowMapStore) Set(ctx context.Context, key string, val any, ttl time.Duration, opts ...cachecore.CallOption) error {
	_ = ctx
	_ = opts
	if ttl < 0 {
		return fmt.Errorf("ttl must be >= 0")
	}
	item := mapItem{value: val}
	if ttl > 0 {
		item.expireAt = time.Now().Add(ttl)
	}
	s.mu.Lock()
	s.data[key] = item
	s.mu.Unlock()
	return nil
}

func (s *SlowMapStore) GetWithTTL(ctx context.Context, key string, opts ...cachecore.CallOption) (any, time.Duration, error) {
	v, err := s.Get(ctx, key, opts...)
	if err != nil {
		return nil, 0, err
	}
	s.mu.RLock()
	item := s.data[key]
	s.mu.RUnlock()
	if item.expireAt.IsZero() {
		return v, 0, nil
	}
	return v, time.Until(item.expireAt), nil
}

func (s *SlowMapStore) Delete(ctx context.Context, key string, opts ...cachecore.CallOption) error {
	_ = ctx
	_ = opts
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	return nil
}

func (s *SlowMapStore) Clear(ctx context.Context) error {
	_ = ctx
	s.mu.Lock()
	s.data = make(map[string]mapItem)
	s.mu.Unlock()
	return nil
}

func (s *SlowMapStore) StoreName() string { return s.name }

var _ cachecore.Store = (*SlowMapStore)(nil)
