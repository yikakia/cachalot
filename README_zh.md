# cachalot

一个面向 Golang 的通用缓存门面库，支持开箱即用，也支持深度扩展。

[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/yikakia/cachalot/actions/workflows/ci.yml/badge.svg)](https://github.com/yikakia/cachalot/actions/workflows/ci.yml)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/yikakia/cachalot?tab=doc)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yikakia/cachalot)

📘 English docs: [README.md](README.md)

## 项目定位

`cachalot` 是一个基于 Go 泛型的缓存库，核心目标是把 m 种缓存实现与 n 种缓存使用模式的组合复杂度，从 **O(m*n)** 降到 **O(m+n)**。
它不是某个具体缓存系统，而是把常见缓存模式抽象成可复用、可组合的能力。

## 当前状态

项目当前以获取早期反馈为目标，暂不承诺 API 冻结。
在正式版发布前，API 和默认行为仍可能调整。

## 特性

- 单级缓存（Single Cache），支持可插拔 `Store`。
- 多级缓存（Multi Cache），支持可配置 fetch / write-back 策略。
- 基于 decorator 的能力编排（codec、逻辑过期、singleflight、miss loader）。
- 统一观测能力（metrics + logger）。
- 同时提供 Builder 高层 API（开箱即用）与 core 低层 API（精细编排）。

## 安装

```bash
go get github.com/yikakia/cachalot
```

## 快速开始（单缓存）

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

## 运行示例

完整可运行示例见 [examples](examples)：

- [examples/01_basic](examples/01_basic)
- [examples/02_codec](examples/02_codec)
- [examples/03_logical_expiry](examples/03_logical_expiry)
- [examples/04_observability](examples/04_observability)
- [examples/05_advanced_multi_cache](examples/05_advanced_multi_cache)

## 选择合适的 API

- `Builder API`（推荐多数场景）：开箱即用，默认集成常用能力。
- `core` 包（高级定制）：需要精确控制工厂、装饰器链路和多级策略时使用。

### Builder（常规使用）

#### 单缓存 Builder：`NewBuilder`

常用能力：

- `WithCacheMissLoader`：未命中时回源。
- `WithCacheMissDefaultWriteBackTTL`：回源成功后的默认回写 TTL。
- `WithSingleFlight`：并发请求合并。
- `WithCodec`：面向字节型存储的编解码。
- `WithLogicExpire*`：逻辑过期（stale-while-revalidate）。
- `WithLogger` / `WithMetrics`：接入观测能力。

#### 多级缓存 Builder：`NewMultiBuilder`

常用能力：

- `WithLoader`：所有层都 miss 时的回源函数（默认对同 key 启用 singleflight，可通过 `WithSingleflightLoader(false)` 关闭）。
- `WithFetchPolicy`：自定义多级缓存的探测顺序与加载策略。
- `WithWriteBack` / `WithWriteBackFilter`：自定义回写行为和目标层过滤规则。
- `WithErrorHandling`：控制回写失败时的 strict / tolerant 策略。

### core（高级编排）

如需完全掌控链路，可直接使用：

- `core/cache`：单缓存抽象（`Cache`、`Store`、Option、Factory/Decorator）。
- `core/multicache`：多级缓存编排（`Config`、策略函数、错误处理）。
- `core/decorator`：可复用能力装饰器。

## 架构说明

单缓存核心架构分为三层：

- `Decorator`：定义“如何用缓存”（并发收敛、回源、防击穿、观测等）。
- `Factory`：将 `Store` 适配为 `Cache[T]`，并处理类型桥接。
- `Store`：封装具体存储客户端并提供统一读写语义。

Builder 默认装配顺序与装饰器执行模型见：

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## 文档导航

- TODOs: [TODO.md](TODO.md)
- 架构与装饰器链路：[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- 常见问题（建设中）：[docs/FAQ.md](docs/FAQ.md)
- 贡献指南（建设中）：[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md)

## 测试与 Mock

仓库使用 `mockgen` 生成接口 Mock 并用于单测：

- `internal/mocks/mock_cache.go` 对应 `cache.Cache[T]`
- `internal/mocks/mock_store.go` 对应 `cache.Store`

重新生成 Mock：

```bash
go generate ./internal/mocks
```

运行测试：

```bash
go test ./...
```

或运行完整脚本（包含子模块）：

```bash
./run_tests.sh
```

## License

© yikakia, 2026~time.Now()

基于 [MIT License](LICENSE) 开源。
