package telemetry

import (
	"context"
)

type Metrics interface {
	Record(context.Context, *Event) error
}

func NoopMetrics() Metrics {
	return &noopMetrics{}
}

type noopMetrics struct{}

func (n *noopMetrics) Record(ctx context.Context, event *Event) error { return nil }
