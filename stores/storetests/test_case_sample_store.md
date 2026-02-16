# `cache.Store` 接口单测模板

## 通用约定

- 所有实现需要通过下列所有 Case
- 使用 `assert` `require` 进行断言
- 每个方法独立的 `t.Run` 子测试
- Store 是对 client 的统一封装，默认需要实现对 context 的支持

## 接口定义
```go
// Store 存储层抽象，对接对象缓存 (ristretto) 字节缓存 (redis)
type Store interface {
	Get(ctx context.Context, key string, opts ...Option ) (any, error)
	// 该键值将在 ttl 时间后过期，如果 ttl 为 0，表示永不过期，如果 ttl 为负数，会返回错误 ErrInvalidTTL
	Set(ctx context.Context, key string, val any, ttl time.Duration, opts ...Option ) error
	// 如果永不过期，ttl 返回 0
	GetWithTTL(ctx context.Context, key string, opts ...Option ) (any, time.Duration, error)
	Delete(ctx context.Context, key string, opts ...Option) error
	Clear(ctx context.Context) error
	StoreName() string
}

```

---

## 1. `Get(ctx, key, opts...) (any, error)`

| Case | 描述 |
|------|------|
| `Get_ExistingKey` | 获取已存在的 key，返回正确的值，error 为 nil |
| `Get_NonExistingKey` | 获取不存在的 key，返回 `cache.ErrNotExist` |
| `Get_ExpiredKey` | 获取已过期的 key（TTL 已到期），返回 `cache.ErrNotExist` |
| `Get_WithOptions` | 使用自定义 `GetOption`，验证 option 被正确传递 |

---

## 2. `Set(ctx, key, val, ttl, opts...) error`

| Case | 描述 |
|------|------|
| `Set_NewKey` | 设置新 key，后续 Get 能获取到正确的值 |
| `Set_OverwriteExistingKey` | 覆盖已存在的 key，新值生效 |
| `Set_ZeroTTL` | TTL 为 0 时的行为（永不过期 ） |
| `Set_NegativeTTL` | TTL 为负数时的行为（应返回错误或忽略） |
| `Set_WithOptions` | 使用自定义 `SetOption`，验证 option 被正确传递 |

---

## 3. `GetWithTTL(ctx, key, opts...) (any, time.Duration, error)`

| Case | 描述 |
|------|------|
| `GetWithTTL_ExistingKey` | 获取已存在的 key，返回正确的值和剩余 TTL |
| `GetWithTTL_NonExistingKey` | 获取不存在的 key，返回 `cache.ErrNotExist` |
| `GetWithTTL_TTLDecreasing` | 验证返回的 TTL 随时间递减（sleep 后再次获取，TTL 应减少） |
| `GetWithTTL_NoExpiry` | 对于永不过期的 key，TTL 应返回 0 |
| `GetWithTTL_WithOptions` | 使用自定义 `GetOption`，验证 option 被正确传递 |

---

## 4. `Delete(ctx, key, opts...) error`

| Case | 描述 |
|------|------|
| `Delete_ExistingKey` | 删除已存在的 key，后续 Get 返回 `ErrNotExist` |
| `Delete_NonExistingKey` | 删除不存在的 key，不应报错（幂等操作） |
| `Delete_WithOptions` | 使用自定义 `DeleteOption`，验证 option 被正确传递 |

---

## 5. `Clear(ctx) error`

| Case | 描述 |
|------|------|
| `Clear_EmptyStore` | 空 store 调用 Clear 不应报错 |
| `Clear_NonEmptyStore` | 清空后所有 key 都应返回 `ErrNotExist` |
| `Clear_MultipleTimes` | 多次调用 Clear 不应报错 |
---

## 6. `StoreName() string`

| Case | 描述 |
|------|------|
| `StoreName_NotEmpty` | 返回非空字符串 |
| `StoreName_Consistent` | 多次调用返回相同的名称 |

---