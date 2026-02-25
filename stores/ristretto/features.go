package ristretto

import (
	"github.com/yikakia/cachalot/core/cache"
)

func WithSynchronousSet(enable bool) cache.CallOption {
	return func(cfgs *cache.CallOptConfig) {
		features := loadOrInitSetFeatures(cfgs)
		features.flush = enable
		features.apply(cfgs)
	}
}

func WithCost(cost int64) cache.CallOption {
	return func(cfgs *cache.CallOptConfig) {
		features := loadOrInitSetFeatures(cfgs)
		features.cost = cost
		features.apply(cfgs)
	}
}

const (
	featureNameSet = "ristretto_set"
)

type setFeatures struct {
	flush bool
	cost  int64
}

func (s *setFeatures) apply(cfg *cache.CallOptConfig) {
	cfg.SetCustomField(featureNameSet, s)
}

func loadOrInitSetFeatures(cfg *cache.CallOptConfig) *setFeatures {
	f, _ := cfg.GetCustomField(featureNameSet)
	if f != nil {
		// assert no conflict
		return f.(*setFeatures)
	}
	return &setFeatures{}
}
