# 决策矩阵 (Decision Matrix)

为了帮助开发者根据实际需求选择最合适的缓存配置，下表列出了常见场景的推荐组合。

| 场景需求 | 存储 (Store) | 阶段特性 (Staged Features) | 增强能力 (Decorators) | 推荐理由 |
| :--- | :--- | :--- | :--- | :--- |
| **极速本地缓存** | `ristretto` / `freecache` | 无 (No Codec) | `Singleflight` | 本地存储直接存取对象，无需序列化，延迟最低。 |
| **分布式大对象** | `redis` / `valkey` | `JSON/Protobuf` + `Zstd/Gzip` | `Singleflight` + `Loader` | 远端存储必须字节化；压缩可大幅减少网络 IO 和存储成本。 |
| **读多写少热点数据** | `L1 (Ristretto) + L2 (Redis)` | `L2: Codec` | `FetchPolicy` + `WriteBack` | 多级缓存平衡了延迟与容量；回写策略确保 L1 快速预热。 |
| **高并发防击穿 (热点刷新)** | 任意 | 无或根据存储决定 | `LogicExpire` + `Singleflight` | 利用逻辑过期 (SWB) 彻底消除物理过期瞬间的请求毛刺。 |
| **缓存穿透防护** | 任意 | 无 | `NilCache` | 缓存空结果，防止无效请求持续压垮数据库。 |

## 选择建议

1. **如果你不确定：**
   使用 `NewBuilder[T](...)` 并开启 `.WithSingleflight()`，这是最稳健的起点。

2. **如果你的存储是远程的（如 Redis）：**
   务必配置 `.WithCodec(codec.JSONCodec{})`（或 Protobuf）。

3. **如果你对性能有极致要求：**
   - 尽量使用本地缓存。
   - 避免不必要的压缩和复杂的装饰器。

4. **如果你的后端加载逻辑非常重：**
   - 开启 `.WithSingleflight()` 以收敛并发。
   - 考虑 `.WithLogicExpire*` 以异步刷新。
