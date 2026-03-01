package storetests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yikakia/cachalot/core/cache"
)

type Config struct {
	WaitingAfterWrite func(*testing.T, cache.Store)
	SetOptions        []cache.CallOption
	EncodeSetValue    func(string) any
	AssertValue       func(t *testing.T, got any, expected string)
}

type Option interface {
	Apply(*Config)
}

type OptionFunc func(*Config)

func (f OptionFunc) Apply(c *Config) {
	f(c)
}

func WithWaitingAfterWrite(f func(t *testing.T, s cache.Store)) Option {
	return OptionFunc(func(c *Config) {
		c.WaitingAfterWrite = f
	})
}

func WithSetOptions(opts ...cache.CallOption) Option {
	return OptionFunc(func(c *Config) {
		c.SetOptions = opts
	})
}

func WithEncodeSetValue(f func(string) any) Option {
	return OptionFunc(func(c *Config) {
		c.EncodeSetValue = f
	})
}

func WithAssertValue(f func(t *testing.T, got any, expected string)) Option {
	return OptionFunc(func(c *Config) {
		c.AssertValue = f
	})
}

func NewConfig() *Config {
	return &Config{
		WaitingAfterWrite: func(*testing.T, cache.Store) {},
		SetOptions:        []cache.CallOption{},
		EncodeSetValue: func(v string) any {
			return v
		},
		AssertValue: func(t *testing.T, got any, expected string) {
			assert.Equal(t, expected, got)
		},
	}
}
