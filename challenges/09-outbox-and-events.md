# 第 09 关：事件契约与 Transactional Outbox

## 业务背景

如果先提交数据库再发 Kafka，进程可能在两步之间崩溃；如果先发 Kafka 再写数据库，又可能出现不存在的业务事件。Outbox 把业务写和待发布事件放进同一个数据库事务。

## 第一部分：事件 Envelope

文件：`internal/messaging/event.go`

### `NewEvent(...)`

- 校验非空 ID、类型、aggregate ID 和非零时间。
- 使用 `json.Marshal` 编码 data，失败时包装 `ErrInvalidEvent`。
- 设置 schema version 为 1，保留 UTC 时间。

### `Event.Validate()`

验证所有必需 envelope 字段、版本为正、`Data` 是合法 JSON。事件类型建议符合 `<domain>.<action>.v<version>`，不要在本关写过度复杂的正则。

## 第二部分：Outbox 表与写入

创建 `migrations/000002_outbox.up.sql/down.sql`。结构以数据模型文档为准，关键索引为 `(status, available_at, created_at)`。

把活动预约和奖励领取改为：同一事务写业务表与 `outbox_events`。为此可以定义：

```go
type UnitOfWork interface {
    WithinTransaction(ctx context.Context, fn func(Repositories) error) error
}
```

不要把 pgx transaction 类型泄露到领域层。

## 第三部分：发布器

实现 Outbox Publisher：

- 批量 `FOR UPDATE SKIP LOCKED` 领取到期记录。
- 发布到 Kafka 后标为 PUBLISHED。
- 失败增加 attempt，按指数退避更新 available_at。
- 错误摘要截断、脱敏；达到阈值标 FAILED 并产生指标。
- 发布成功但状态更新前崩溃会重复发布，这是协议允许的。

创建 `docs/adr/0003-transactional-outbox.md`，明确 at-least-once 语义。

## 需要补充的测试

- 不可 JSON 编码的数据（如 channel）返回 `ErrInvalidEvent`。
- 业务写失败时 outbox 也不存在。
- 两个 publisher 并发不会同时锁住同一 pending row。
- 发布后崩溃模拟会产生重复消息，而不是丢消息。

## 涉及知识

- 双写问题、事务边界、至少一次投递。
- `SKIP LOCKED`、批处理、指数退避和抖动。
- 事件 schema 演进和不可变事件。

## 通关测试

```powershell
go test ./internal/messaging -v
go test ./coursechecks -run TestLevel09MessagingArtifacts -v
go test -tags=integration ./tests/integration -run Outbox -v
```

## 完成标准

- 单元、静态与 Outbox 集成测试通过。
- 能解释为什么 Outbox 没有提供 exactly-once，以及为什么这并不可怕。
