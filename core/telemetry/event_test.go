package telemetry

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestAddCustomFields(t *testing.T) {
	t.Parallel()

	evt := &Event{}
	ctx := ContextWithEvent(context.Background(), evt)

	var wg sync.WaitGroup
	count := 1000

	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			defer wg.Done()
			AddCustomFields(ctx, map[string]string{
				fmt.Sprintf("key-%d", idx): fmt.Sprintf("val-%d", idx),
			})
		}(i)
	}

	wg.Wait()

	fields := evt.FrozenCustomFields()
	if len(fields) != count {
		t.Errorf("expected %d fields, got %d", count, len(fields))
	}
}
