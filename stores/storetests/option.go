package storetests

import (
	"github.com/yikakia/cachalot/core/cache"
)

type Config struct {
	WaitingAfterWrite func(cache.Store)
	SetOptions        []cache.CallOption
}

type Option interface {
	Apply(*Config)
}

type OptionFunc func(*Config)

func (f OptionFunc) Apply(c *Config) {
	f(c)
}

func WithWaitingAfterWrite(f func(cache.Store)) Option {
	return OptionFunc(func(c *Config) {
		c.WaitingAfterWrite = f
	})
}

func WithSetOptions(opts ...cache.CallOption) Option {
	return OptionFunc(func(c *Config) {
		c.SetOptions = opts
	})
}

func NewConfig() *Config {
	return &Config{
		WaitingAfterWrite: func(cache.Store) {},
		SetOptions:        []cache.CallOption{},
	}
}
