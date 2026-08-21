# 第 12 关：健康检查、优雅停机与可观测性

## 业务背景

“能启动”不等于“能在 Kubernetes 中可靠运行”。进程必须准确报告能否接流量，在 SIGTERM 后排空，同时用 metrics、logs、traces 给出排障证据。

## 第一部分：就绪注册表

文件：`internal/platform/health/health.go`

### `NewRegistry(checks)`

复制输入 map；不能让调用方之后的修改影响 registry。

### `SetDraining(value)`

用锁保护排空状态。

### `Ready(ctx)`

- draining 时立即 not ready，并返回 `draining:true`。
- 对每个 check 返回 `ok` 或 `unavailable`，不能回传底层秘密错误。
- 任一关键检查失败则 overall false。
- 复制检查函数后释放锁再调用，避免慢依赖长时间占锁。

## 第二部分：API/Worker 进程装配

完成 `cmd/api/main.go`、`cmd/worker/main.go`：

```text
load config -> build logger/tracer/dependencies -> start server/consumer
SIGTERM -> mark not ready -> wait propagation -> shutdown with timeout
        -> finish current jobs -> close Kafka/Redis/DB -> flush telemetry -> exit
```

实现 `/health/live`、`/health/ready`、内部 `/metrics`。liveness 不检查数据库；readiness 根据实际路由的关键依赖选择检查项。

## 第三部分：三大信号

创建 `docs/observability.md`，定义：

- HTTP RED：request rate、error rate、duration（含 P50/P95/P99）。
- Reward：成功、重复、库存耗尽、库存偏差。
- Kafka：处理量、失败、重试、DLQ、consumer lag。
- DB pool：使用、idle、等待、query duration。
- trace：HTTP → PostgreSQL/Redis → Kafka publish → consume → DB。
- 结构化日志字段与脱敏规则。

指标 label 不允许 user/event/request UUID；路由使用模板而非原始 URL。

## 必须补充的测试

- 构造后修改原 checks map，不影响 registry。
- check 尊重 context timeout。
- `httptest.Server` 验证 live/ready 状态码。
- 启动真实 API 子进程发送 SIGTERM，验证在宽限期内退出（E2E）。

## 涉及知识

- liveness/readiness/startup 的职责。
- Go HTTP graceful shutdown、signal.NotifyContext。
- RED/USE、直方图、指标基数。
- W3C Trace Context 和异步 span link/parent。

## 通关测试

```powershell
go test -race ./internal/platform/health -count=20
go test ./coursechecks -run TestLevel12ObservabilityArtifacts -v
go test -tags=e2e ./tests/e2e -run 'Health|Shutdown' -v
```

## 完成标准

- 健康、排空与 E2E 测试通过。
- 能解释“DB down 时 live=200、ready=503”为什么合理。
- Grafana 或本地 collector 能关联一次领取请求的日志和 trace。
