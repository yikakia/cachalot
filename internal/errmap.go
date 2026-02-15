package internal

import (
	"errors"

	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/telemetry"
)

// ResultFromErr 提供自定义的 notFoundErr 将err转化为result
// 为什么要放到一个单独的包？为了避免 cache -> telemetry -> cache 的循环依赖
func ResultFromErr(err error) telemetry.Result {
	switch {
	case err == nil:
		return telemetry.ResultHit
	case errors.Is(err, cache.ErrNotFound):
		return telemetry.ResultMiss
	default:
		return telemetry.ResultFail
	}
}
