package telemetry

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"
)

func ContextWithEvent(ctx context.Context, evt *Event) context.Context {
	return context.WithValue(ctx, eventKey{}, evt)
}

func AddCustomFields(ctx context.Context, fields map[string]string) {
	if len(fields) == 0 {
		return
	}
	value := ctx.Value(eventKey{})
	if value == nil {
		return
	}
	e := value.(*Event)
	e.mu.Lock()
	defer e.mu.Unlock()

	for k, v := range fields {
		e.getOrInitCustomFields()[k] = v
	}
}

type eventKey struct{}

type Event struct {
	Op Op
	// Get GetWithTTL 的结果 hit miss fail
	// 可以使用 ResultFromErr 进行简单转换
	Result    Result
	CacheName string
	StoreName string
	Latency   time.Duration
	Error     error // 最后拿到的err

	mu           sync.Mutex
	customFields map[string]string
}

func (e *Event) getOrInitCustomFields() map[string]string {
	if e.customFields == nil {
		e.customFields = map[string]string{}
	}
	return e.customFields
}

func (e *Event) FrozenCustomFields() map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()
	c := e.getOrInitCustomFields()
	return maps.Clone(c)
}

type Op string

const (
	OpGet        Op = "get"
	OpSet        Op = "set"
	OpGetWithTTL Op = "get_with_ttl"
	OpDelete     Op = "delete"
	OpClear      Op = "clear"
)

type Result string

const (
	ResultHit  = "hit"
	ResultMiss = "miss"
	ResultFail = "fail"
)

// ResultFromErr 提供自定义的 notFoundErr 将err转化为result
func ResultFromErr(err error, notFoundErr error) Result {
	switch {
	case err == nil:
		return ResultHit
	case errors.Is(err, notFoundErr):
		return ResultMiss
	default:
		return ResultFail
	}
}
