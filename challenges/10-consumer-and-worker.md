# 第 10 关：Kafka 消费幂等与有界 Worker Pool

## 业务背景

Worker 可能在业务写成功、offset 提交前崩溃，因此同一消息会再次到达。消费端必须识别重复；并发数必须有上限，停机时还要响应 context。

## 第一部分：消费者 Processor

文件：`internal/consumer/processor.go`

### `Processor.Process(ctx, event)`

- 先调用 `event.Validate()`，无效事件不进入 handler。
- 校验 processor name、store、handler 已配置；建议让构造函数返回 error。
- 调用 `store.ExecuteOnce(eventID, consumerName, fn)`。
- fn 内调用 handler，任何错误原样包装返回。
- duplicate 返回 `processed=false, err=nil`。

PostgreSQL 的 `DedupStore` 必须在一个事务里：插入 `(event_id,consumer_name)` → 业务副作用 → commit。handler 失败必须回滚 dedup 标记。

## 第二部分：Worker Pool

文件：`internal/worker/pool.go`

### `Run(ctx, workerCount, jobs, handler) error`

- workerCount <= 0 返回 `ErrInvalidWorkerCount`。
- 创建派生 context 和 cancel。
- 通过有界 job channel 启动固定数量 worker。
- 任意 handler 首次失败时记录错误并 cancel；不能发生 send on closed channel。
- 等待所有已启动 goroutine 退出后返回第一个错误。
- 调用者取消时返回 context error（如果没有更早业务错误）。

可以使用标准库，也可以在理解原理后使用 `errgroup`。不允许每个 job 无限制启动 goroutine。

## 第三部分：Kafka 接线

- 选择 Kafka 客户端并固定版本。
- 消费成功或重复后再提交 offset。
- 临时错误有界重试；不可重试错误和耗尽错误进入 `fanpulse.dlq.v1`。
- 传播 trace context；日志包含 event ID，不把它放进 Prometheus label。

## 必须补充的测试

- handler 失败后 dedup 标记未提交，第二次可以重试。
- 相同 event 给不同 consumer name，各自处理一次。
- worker handler 阻塞时取消 context，测试在短超时内结束。
- `go test -race -count=50` 无数据竞争或泄漏症状。

## 涉及知识

- Kafka offset、consumer group、partition 并行度。
- context cancellation、channel ownership、WaitGroup。
- first-error wins、goroutine 泄漏。
- DLQ 不是垃圾桶：需要告警、检查和重放流程。

## 通关测试

```powershell
go test -race ./internal/consumer ./internal/worker -count=30
go test -tags=integration ./tests/integration -run 'Consumer|Kafka' -v
```

## 完成标准

- 重复投递只产生一次副作用。
- Worker 并发永远不超过配置值，取消和错误路径都退出。
- 能口述“DB commit 成功但 offset commit 失败”后的完整恢复过程。
