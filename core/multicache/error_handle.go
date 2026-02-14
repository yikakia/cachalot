package multicache

type ErrorHandleMode int

const (
	ErrorHandleStrict   ErrorHandleMode = iota // 严格模式 当 WriteBackFn 返回 err 时，Get 也直接返回 err
	ErrorHandleTolerant                        // 容忍模式 当 WriteBackFn 返回 err 时，吞掉 err 仅保留日志
)
