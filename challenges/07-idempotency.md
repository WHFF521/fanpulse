# 第 07 关：Idempotency-Key 与并发重放

## 业务背景

服务可能已经成功，但响应在网络中丢失。客户端用同一个 key 重试时，应得到同一个资源；若同一个 key 被用于不同请求，则必须冲突。

## 需要完成的代码

文件：`internal/idempotency/idempotency.go`

### `NewMemoryStore()`

初始化 map。这个实现只用于学习和单进程测试，第 08 关会用 Redis 替换。

### `Execute(ctx, key, requestHash, fn)`

语义：

- 空 key 返回 `ErrEmptyKey`，不得执行 fn。
- 第一个调用者登记 processing entry，释放全局锁后执行 fn。
- 相同 key/hash 的并发调用者等待 `done` 或自己的 context 取消。
- fn 完成后保存 result/error、关闭 `done`；等待者返回 `replayed=true`。
- 已有 key 但 hash 不同，返回 `ErrKeyReused`。
- 绝不能在持有全局 mutex 时运行 fn。

设计选择：失败是否缓存？本关测试允许缓存结果；你需要在学习日志区分“确定性业务失败”和“临时基础设施失败”，并为生产策略补测试。

## 需要补充的测试

- 空 key 不调用 fn。
- 等待中的 context 取消会及时返回。
- fn 返回错误时所有等待者都结束，不发生死锁。
- 用 `go test -race -count=100` 验证。

## 涉及知识

- mutex、channel close 广播、happens-before。
- single-flight、response replay、request hash。
- context cancellation 与 goroutine 泄漏。
- 业务唯一约束与短期幂等缓存的双层保护。

## 通关测试

```powershell
go test -race ./internal/idempotency -count=50
```

## 常见错误

- 在锁内运行 fn，导致所有 key 串行。
- 两个 goroutine 都认为自己是 owner。
- 错误路径忘记 close channel，等待者永久阻塞。
- hash 直接包含秘密或把原始 key 写日志。

## 完成标准

- 现有与新增并发测试稳定通过。
- 能画出 owner、waiter 和重放结果的时序图。
