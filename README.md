# cachalot

A universal cache facade library for Go, supporting both out-of-the-box usage and deep customization.

[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/yikakia/cachalot/actions/workflows/ci.yml/badge.svg)](https://github.com/yikakia/cachalot/actions/workflows/ci.yml)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/yikakia/cachalot?tab=doc)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yikakia/cachalot)

📘 中文文档请查看: [README_zh.md](README_zh.md)

## Project Positioning

`cachalot` is a generic cache library based on Go generics. The core goal is to reduce the combinatorial complexity of m cache implementations with n cache usage patterns from **O(m*n)** to **O(m+n)**.
It is not a specific cache system, but rather abstracts common caching patterns into reusable and composable capabilities.

## Project Status

This project is currently focused on gathering early feedback and is **not production-ready yet**.
The API is not frozen, and APIs/default behaviors may still change before the first stable release.
Feedback is highly welcome on:

- API naming and semantics.
- Usage ergonomics in real service flows.
- Default strategy implementations (fetch/write-back/singleflight).
- Extension experience for custom store/factory/decorator integrations.

## Comparison with eko/gocache

Both `cachalot` and `eko/gocache` aim to simplify cache access, but they optimize for different things:

- `eko/gocache` focuses on a unified facade with ready-to-use adapters for fast adoption.
- `cachalot` focuses on structured layering with explicit `Store`, `Factory`, and `Decorator` boundaries.

This separation makes it easier to compose features by evolving storage implementation, construction/type-bridging logic, and usage policies independently.

## Features

- Single-level cache with pluggable `Store` implementations.
- Multi-level cache with customizable fetch and write-back policies.
- Decorator-based capability orchestration (codec, logical expiry, singleflight, miss loader).
- Unified observability (metrics + logger).
- Both high-level Builder API (out-of-the-box usage) and low-level core API (fine-grained orchestration).

## Install

```bash
go get github.com/yikakia/cachalot
```

## Quick Start (Single Cache)

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

u, err := cache.Get(context.Background(), "user-1")
```

## Running Examples

For complete runnable examples, see [examples](examples):

- [examples/01_basic](examples/01_basic)
- [examples/02_codec](examples/02_codec)
- [examples/03_logical_expiry](examples/03_logical_expiry)
- [examples/04_observability](examples/04_observability)
- [examples/05_advanced_multi_cache](examples/05_advanced_multi_cache)

## Choosing the Right API

- `Builder API` (recommended for most scenarios): Out-of-the-box, with commonly-used features integrated by default.
- `core` package (advanced customization): Use when you need precise control over factories, decorator chains, and multi-level policies.

### Builder API (Regular Usage)

#### Single-cache Builder: `NewBuilder`

Common options:

- `WithCacheMissLoader`: Load from origin when a key is missed.
- `WithCacheMissDefaultWriteBackTTL`: Default write-back TTL after loader returns successfully.
- `WithSingleFlight`: Merge concurrent requests.
- `WithCodec`: Codec for byte-oriented stores.
- `WithLogicExpire*`: Logical expiration (stale-while-revalidate).
- `WithLogger` / `WithMetrics`: Observability integration.

#### Multi-cache Builder: `NewMultiBuilder`

Common options:

- `WithLoader`: Fallback loader when all levels miss (singleflight for the same key is enabled by default; disable via `WithSingleflightLoader(false)`).
- `WithFetchPolicy`: Customize probing order and load strategy across levels.
- `WithWriteBack` / `WithWriteBackFilter`: Control write-back behavior and target-level filtering rules.
- `WithErrorHandling`: Control strict/tolerant behavior for write-back failures.

### core (Advanced Orchestration)

For full control over the pipeline, use:

- `core/cache`: Single-cache abstractions (`Cache`, `Store`, Option, Factory/Decorator).
- `core/multicache`: Multi-level cache orchestration (`Config`, policy functions, error handling).
- `core/decorator`: Reusable capability decorators.

## Architecture

The single-cache core architecture is divided into three layers:

- `Decorator`: Defines "how to use cache" (concurrent deduplication, load-through, cache penetration protection, observability, etc.).
- `Factory`: Adapts `Store` to `Cache[T]` and handles type bridging.
- `Store`: Encapsulates the concrete storage client and provides unified read/write semantics.

The default assembly order of Builder and the decorator execution model are described in:

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## Documentation Navigation

- TODOs: [TODO.md](TODO.md)
- Architecture and decorator chain: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- FAQ (under construction): [docs/FAQ.md](docs/FAQ.md)
- Contributing Guide (under construction): [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)

## Testing and Mocks

This repo uses `mockgen` to generate interface mocks for unit tests:

- `internal/mocks/mock_cache.go` for `cache.Cache[T]`
- `internal/mocks/mock_store.go` for `cache.Store`

Regenerate mocks with:

```bash
go generate ./internal/mocks
```

Run tests:

```bash
go test ./...
```

Or run the full script (including submodules):

```bash
./run_tests.sh
```

## License

© yikakia, 2026~time.Now()

Released under the [MIT License](LICENSE).
