# 第 08 关：Token Bucket、Redis 与奖励快速路径

## 业务背景

活动开场流量不能全部打到 PostgreSQL。限流先保护服务，Redis Lua 再把“查重复、查库存、扣减”合成一个原子操作。Redis 仍不是最终事实源。

## 第一部分：本地 Token Bucket

文件：`internal/ratelimit/bucket.go`

### `NewBucket(capacity, refillPerSecond, now)`

创建一个初始装满的桶。建议将非法 capacity/refill 改为显式错误；若保持当前签名，至少保证不会 panic 或产生 NaN。

### `Bucket.Allow(now) bool`

在锁内：

1. 计算 `elapsed := now.Sub(last).Seconds()`，时间倒退按 0 处理。
2. `tokens = min(capacity, tokens + elapsed*refillRate)`。
3. 更新 last；若 token >= 1 则扣 1 并允许。

禁止使用真实 sleep 写测试；测试传入虚拟时间。

## 第二部分：Redis 实现

使用 go-redis 创建：

```go
type Limiter interface {
    Allow(ctx context.Context, subject, route string, now time.Time) (Decision, error)
}

type Decision struct {
    Allowed    bool
    RetryAfter time.Duration
    Remaining int
}
```

再创建 `internal/storage/redis/claim.lua`，一次完成：

- 判断用户是否已领取。
- 判断 stock 是否大于 0。
- 登记用户并扣减库存。
- 返回稳定结果码：`CLAIMED`、`DUPLICATE`、`OUT_OF_STOCK`。

脚本契约已经由 `tests/integration/redis_test.go` 固定：`KEYS[1]` 是库存 key，`KEYS[2]` 是已领取用户集合，`ARGV[1]` 是用户 ID；返回值必须是上面的字符串之一。

所有 key 使用环境前缀；Cluster 下相关 key 使用相同 `{reward_id}` hash tag。写集成测试并发调用脚本，验证不超发。

## 第三部分：一致性边界

创建 `docs/adr/0002-use-redis-fast-path.md`，明确：

- PostgreSQL 是事实源，Redis 可以重建。
- Redis 成功而持久化未完成时，HTTP 只能返回 202/PENDING。
- 提供核对任务：比较 DB 成功 claim、DB remaining 与 Redis stock。
- Redis 不可用时选择数据库基线或受控 503，不能静默放行。

## 涉及知识

- Token Bucket 数学、浮点边界、时钟注入。
- Redis Lua 原子性与 Cluster slot。
- Cache/coordination data 与 source of truth。
- backpressure、429、`Retry-After`。

## 通关测试

```powershell
go test -race ./internal/ratelimit -count=30
go test ./coursechecks -run TestLevel08RedisArtifacts -v
go test -tags=integration ./tests/integration -run Redis -count=3 -v
```

## 完成标准

- 并发限流和 Lua 库存测试稳定通过。
- 能解释为什么 Redis 脚本成功不等于奖励已经永久发放。
- 为时间倒退、零 refill、不同用户补测试。
