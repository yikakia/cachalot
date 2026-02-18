# cachalot

一个面向 Golang 的通用缓存门面库，既支持开箱即用，也支持深度扩展。

[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-blue?logo=go&logoColor=white)](https://pkg.go.dev/github.com/yikakia/cachalot?tab=doc)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yikakia/cachalot)

📘 English docs: [README.md](README.md)

## 项目定位

`cachalot` 是一个基于 Go 泛型的缓存库，聚焦于封装缓存系统中的通用模式，并允许灵活组合：

- 单级缓存（Single Cache），可插拔 Store。
- 多级缓存（Multi Cache），支持可配置的 fetch/write-back 策略。
- 基于 decorator 的能力编排（codec、逻辑过期、singleflight、miss loader）。
- 统一观测能力（metrics + logger）。
- 对常规用户提供 Builder，对高级用户开放 core 编排能力。

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

u, err := cache.Get(context.Background(), "1")
```

可运行示例见：

- [examples/single](examples/single)
- [examples/multi](examples/multi)

## 常规用户：Builder 用法

### 单缓存 Builder：`NewBuilder`

常用能力：

- `WithCacheMissLoader`：未命中时回源。
- `WithCacheMissDefaultWriteBackTTL`：回源后的默认回写 TTL。
- `WithSingleFlight`：并发请求合并。
- `WithCodec`：面向字节型存储的编解码。
- `WithLogicExpire*`：逻辑过期（stale-while-revalidate）。
- `WithLogger` / `WithMetrics`：接入观测。

### 多级缓存 Builder：`NewMultiBuilder`

常用能力：

- `WithLoader`：多级都 miss 时回源。
- `WithFetchPolicy`：自定义探测顺序/策略。
- `WithWriteBack` / `WithWriteBackFilter`：自定义回写行为。
- `WithErrorHandling`：回写失败时 strict/tolerant 策略。

## 高级用户：基于 core 自定义编排

如需完全掌控缓存流程，可以直接使用 `core` 包：

- `core/cache`：单缓存抽象（`Cache`、`Store`、Option、Factory/Decorator）。
- `core/multicache`：多级缓存编排（`Config`、策略函数、错误处理）。
- `core/decorator`：可复用的特性装饰器。

通过这些能力，你可以实现完全自定义的缓存执行链路，而不受高层默认配置约束。

## 测试与 Mock

仓库使用 `mockgen` 生成接口 Mock 并用于单测：

- `core/cache/mocks/mock_cache.go`：`cache.Cache[T]` 的 mock。
- `core/cache/mocks/mock_store.go`：`cache.Store` 的 mock。

重新生成 mock：

```bash
go generate ./core/cache/mocks
```

运行单测：

```bash
go test ./...
```

## License

基于 [MIT License](LICENSE) 开源。
