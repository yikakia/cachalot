package multicache

import (
	"github.com/yikakia/cachalot/core/telemetry"
)

type Config[T any] struct {
	LoaderFn LoaderFn[T]

	FetchPolicy          FetchPolicy[T]
	WriteBackCacheFilter WriteBackCacheFilter[T]
	WriteBackFn          WriteBackFn[T]
	ErrorHandleMode      ErrorHandleMode
	Observable           *telemetry.Observable
}
