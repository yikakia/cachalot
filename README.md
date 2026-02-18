# cachalot

The cache facade for Golang, with composable features for both quick adoption and deep customization.

[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/yikakia/cachalot?tab=doc)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yikakia/cachalot)

📘 中文文档请查看: [README_zh.md](README_zh.md)

## What is cachalot?

`cachalot` is a generic cache library in Go that wraps common caching patterns and lets you combine them safely:

- Single-level cache with pluggable stores.
- Multi-level cache with customizable fetch and write-back policy.
- Feature composition via decorators (codec, logical expire, singleflight, miss loader).
- Unified telemetry (metrics + logger).
- High-level builders for daily usage and low-level core APIs for advanced orchestration.

## Install

```bash
go get github.com/yikakia/cachalot
```

## Quick start (single cache)

```go
builder, err := cachalot.NewBuilder[User]("user-cache", store)
if err != nil {
	panic(err)
}

cache, err := builder.
	WithCacheMissLoader(loadUser).
	WithCacheMissDefaultWriteBackTTL(time.Minute).
	Build()
if err != nil {
	panic(err)
}

u, err := cache.Get(context.Background(), "1")
```

For runnable samples, see:

- [examples/single](examples/single)
- [examples/multi](examples/multi)

## Builder APIs for regular users

### Single cache builder: `NewBuilder`

Key options:

- `WithCacheMissLoader`: read-through loader when key is missed.
- `WithCacheMissDefaultWriteBackTTL`: write-back TTL after loader returns.
- `WithSingleFlight`: deduplicate concurrent load/get traffic.
- `WithCodec`: serialize values when your store is byte-oriented.
- `WithLogicExpire*`: stale-while-revalidate style logical expiration.
- `WithLogger` / `WithMetrics`: observability integration.

### Multi cache builder: `NewMultiBuilder`

Key options:

- `WithLoader`: fallback loader when all levels miss.
- `WithFetchPolicy`: customize cache probing order/strategy.
- `WithWriteBack` and `WithWriteBackFilter`: control how recovered data is backfilled.
- `WithErrorHandling`: strict/tolerant behavior for write-back failures.

## Extensibility for advanced users

If you need full control of orchestration, use `core` directly:

- `core/cache`: single cache abstractions (`Cache`, `Store`, options, custom factory/decorators).
- `core/multicache`: multi-level cache orchestration primitives (`Config`, policies, filters).
- `core/decorator`: reusable feature decorators that can be assembled manually.

This allows you to build your own cache pipeline without being constrained by high-level defaults.

## Testing and mocks

This repo uses `mockgen` to generate interface mocks for unit tests:

- `core/cache/mocks/mock_cache.go` for `cache.Cache[T]`
- `core/cache/mocks/mock_store.go` for `cache.Store`

Regenerate mocks with:

```bash
go generate ./core/cache/mocks
```

Run tests:

```bash
go test ./...
```

## License

Released under the [MIT License](LICENSE).
