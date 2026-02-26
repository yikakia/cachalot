# Compression

`Compression` 用于在 byte-stage 对缓存值做压缩/解压，降低存储体积与网络带宽。

## 1. API

```go
builder.WithCompression(codec.GzipCompressionCodec{})
```

- 公开调用方式是 Builder Decorator 风格。
- 内部会编译到 byte-stage（不是 behavior decorator）。

## 2. 组合规则

- `T == []byte`：可仅启用 compression，不强制 codec。
- `T != []byte`：需要 `WithCodec(...)` 或 `WithTypeAdapter(...)` 提供 `T <-> []byte` 适配。
- 当 `WithLogicExpire...` 且启用 byte-stage 时，当前要求配置 `WithCodec(...)` 以适配 `LogicTTLValue[T]`。

## 3. 顺序

- `Set`: encode -> compress -> store
- `Get`: load -> decompress -> decode

压缩/解压失败直接返回错误，不做吞错。
