# 可观测性

本文说明 `cachalot` 的可观测性抽象（`telemetry`）以及单级缓存、多级缓存的接入方式。

## 1. 设计目标

`cachalot` 不绑定具体日志和指标实现，通过接口解耦：

- 你可以直接复用已有日志体系（slog/zap 等）。
- 你可以直接复用已有指标体系（Prometheus/OpenTelemetry 等）。
- 库内装饰器和策略通过统一事件模型上报关键行为。

## 2. 核心接口

```go
// core/telemetry/logger.go
type Logger interface {
    DebugContext(ctx context.Context, msg string, args ...any)
    InfoContext(ctx context.Context, msg string, args ...any)
    WarnContext(ctx context.Context, msg string, args ...any)
    ErrorContext(ctx context.Context, msg string, args ...any)
}

// core/telemetry/metrics.go
type Metrics interface {
    Record(context.Context, *Event) error
}

// core/telemetry/observer.go
type Observable struct {
    Metrics
    Logger
}
```

默认实现：

- `telemetry.NoopMetrics()`
- `telemetry.SlogLogger()`

## 3. 事件模型

```go
// core/telemetry/event.go
type Event struct {
    Op        Op
    Result    Result
    CacheName string
    StoreName string
    Latency   time.Duration
    Error     error
    // customFields 通过 AddCustomFields 注入
}
```

关键字段语义：

- `Op`：操作类型，如 `get/set/delete/clear/get_with_ttl`。
- `Result`：主要用于读操作，通常是 `hit/miss/fail`。
- `CacheName` / `StoreName`：用于按缓存实例、存储后端打标签。
- `Latency` / `Error`：用于时延与失败分析。

### 自定义字段

可通过上下文向当前事件写入附加标签：

```go
telemetry.AddCustomFields(ctx, map[string]string{
    "source": "loader",
})
```

例如 `multicache.FetchPolicySequential` 会打 `source=cache_i` 或 `source=loader`。

## 4. 单级缓存如何接入

`Builder` 默认会注入 `decorator.NewObservableDecorator`（见根目录 `cache.go`），并使用 `WithLogger/WithMetrics` 指定实现。

```go
builder, _ := cachalot.NewBuilder[User]("user-cache", store)
c, err := builder.
    WithLogger(myLogger).
    WithMetrics(myMetrics).
    Build()
```

如果使用 low-level API，也可手动注入：

```go
cache.New[T](name, store,
    cache.WithSimpleFactory(func(s cache.Store) (cache.Cache[T], error) {
        return cache.NewBaseCache[T](s), nil
    }),
    cache.WithObservable(&telemetry.Observable{Metrics: myMetrics, Logger: myLogger}),
    cache.WithDecorator(func(inner cache.Cache[T], ob *telemetry.Observable) (cache.Cache[T], error) {
        return decorator.NewObservableDecorator(inner, store.StoreName(), name, ob), nil
    }),
)
```

## 5. 多级缓存如何接入

`MultiBuilder` 默认同样使用 `NoopMetrics + SlogLogger`，可通过：

- `WithLogger(...)`
- `WithMetrics(...)`

覆盖后在 `Build()` 中注入 `cfg.Observable`，并由 `core/multicache/observable.go` 的装饰器上报事件。

```go
mc, err := cachalot.NewMultiBuilder[User]("multi", l1, l2).
    WithLoader(loadUser).
    WithLogger(myLogger).
    WithMetrics(myMetrics).
    Build()
```

## 6. Logger 集成示例

### slog

```go
func SlogLogger() telemetry.Logger {
    return telemetry.SlogLogger()
}
```

### zap

```go
type ZapLogger struct { l *zap.Logger }

func (z *ZapLogger) DebugContext(_ context.Context, msg string, args ...any) { z.l.Sugar().Debugw(msg, args...) }
func (z *ZapLogger) InfoContext(_ context.Context, msg string, args ...any)  { z.l.Sugar().Infow(msg, args...) }
func (z *ZapLogger) WarnContext(_ context.Context, msg string, args ...any)  { z.l.Sugar().Warnw(msg, args...) }
func (z *ZapLogger) ErrorContext(_ context.Context, msg string, args ...any) { z.l.Sugar().Errorw(msg, args...) }
```

## 7. Metrics 集成示例

### Noop

```go
func NoopMetrics() telemetry.Metrics {
    return telemetry.NoopMetrics()
}
```

### Prometheus（示意）

```go
type Metrics struct {
    operationCounter *prometheus.CounterVec
    operationLatency *prometheus.HistogramVec
}

func (m *Metrics) Record(ctx context.Context, event *telemetry.Event) error {
    if event == nil {
        return nil
    }
    m.operationCounter.WithLabelValues(event.CacheName, event.StoreName, string(event.Op), string(event.Result)).Inc()
    m.operationLatency.WithLabelValues(event.CacheName, event.StoreName, string(event.Op)).Observe(event.Latency.Seconds())
    return nil
}
```

## 8. 实践建议

- 指标维度先固定：`cache_name/store_name/op/result`。
- 高频路径避免过多高基数字段，`customFields` 只放必要标签。
- `Tolerant` 策略下要重点监控“回写失败率”，避免静默退化。
- `Metrics.Record` 内部异常不应影响主流程，建议吞错或降级处理。
