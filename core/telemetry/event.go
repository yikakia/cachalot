package telemetry

import (
	"context"
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
	m := e.getOrInitCustomFields()

	e.mu.Lock()
	defer e.mu.Unlock()
	maps.Copy(m, fields)
}

type eventKey struct{}

type Event struct {
	Op Op
	// 当接口为 Get GetWithTTL 时有值 hit miss fail 其他接口置空
	Result    Result
	CacheName string
	StoreName string
	Latency   time.Duration
	Error     error // 最后拿到的err

	mu           sync.Mutex
	fieldOnce    sync.Once
	customFields map[string]string
}

func (e *Event) getOrInitCustomFields() map[string]string {
	e.fieldOnce.Do(func() {
		e.customFields = make(map[string]string)
	})
	return e.customFields
}

func (e *Event) FrozenCustomFields() map[string]string {
	c := e.getOrInitCustomFields()
	e.mu.Lock()
	defer e.mu.Unlock()
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
