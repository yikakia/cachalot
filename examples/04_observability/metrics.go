package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yikakia/cachalot/core/telemetry"
)

type consoleLogger struct{}

func (l *consoleLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	log.Printf("[DEBUG] %s %v", msg, args)
}
func (l *consoleLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	log.Printf("[INFO]  %s %v", msg, args)
}
func (l *consoleLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	log.Printf("[WARN]  %s %v", msg, args)
}
func (l *consoleLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	log.Printf("[ERROR] %s %v", msg, args)
}

type statsMetrics struct {
	mu           sync.Mutex
	total        int
	hits         int
	misses       int
	fails        int
	totalLatency time.Duration
}

func newStatsMetrics() *statsMetrics { return &statsMetrics{} }

func (m *statsMetrics) Record(ctx context.Context, evt *telemetry.Event) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()

	m.total++
	m.totalLatency += evt.Latency
	switch evt.Result {
	case telemetry.ResultHit:
		m.hits++
	case telemetry.ResultMiss:
		m.misses++
	case telemetry.ResultFail:
		m.fails++
	}
	fmt.Printf("metric: op=%s result=%s latency=%s cache=%s store=%s\n", evt.Op, evt.Result, evt.Latency, evt.CacheName, evt.StoreName)

	if evt.Op == telemetry.OpDelete {
		return errors.New("demo metrics sink timeout")
	}
	return nil
}

func (m *statsMetrics) Report() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.total == 0 {
		fmt.Println("no metrics recorded")
		return
	}
	hitRate := float64(m.hits) / float64(m.hits+m.misses)
	avgLatency := m.totalLatency / time.Duration(m.total)
	fmt.Printf("total=%d hits=%d misses=%d fails=%d hit_rate=%.2f avg_latency=%s\n", m.total, m.hits, m.misses, m.fails, hitRate, avgLatency)
}
