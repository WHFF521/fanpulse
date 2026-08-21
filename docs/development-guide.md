# FanPulse README 补充：开发环境指南

## 1. 目标环境

项目以 Linux 容器为运行基准，开发者可使用 Windows + WSL2、macOS 或 Linux。Windows 原生 Shell 也可操作，但 Make、Shell 脚本和容器路径以 WSL2/Linux 为优先验证环境。

建议固定并通过工具文件记录版本，而不是依赖“最新版”：

- Go：项目 `go.mod` 指定版本（建立代码时选当前受支持稳定版）。
- Docker Engine/Desktop 与 Compose v2。
- `golangci-lint`、`migrate` 或 Atlas、`mockgen`（若采用）。
- `kubectl`、`kind`、`helm`。
- `k6`。
- 可选：Terraform、Argo CD CLI、`jq`、`psql`、`redis-cli`、Kafka CLI。

版本升级通过独立 PR 完成，并在 CI 与本地路径验证。

## 2. 配置约定

仓库提交 `.env.example`，本地复制为 `.env`；`.env` 必须被 `.gitignore` 忽略。

预期最小配置：

```dotenv
FANPULSE_ENV=development
FANPULSE_HTTP_ADDR=:8080
FANPULSE_LOG_LEVEL=debug
FANPULSE_DATABASE_URL=postgres://fanpulse:fanpulse@localhost:5432/fanpulse?sslmode=disable
FANPULSE_REDIS_ADDR=localhost:6379
FANPULSE_KAFKA_BROKERS=localhost:9092
FANPULSE_JWT_ISSUER=http://localhost:8081
FANPULSE_JWT_AUDIENCE=fanpulse-api
FANPULSE_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

原则：

- 环境变量以 `FANPULSE_` 开头，配置结构在启动时一次解析并校验。
- 不提交真实密钥；`.env.example` 仅使用本地无风险占位值。
- 测试不得隐式连接开发数据库；集成测试创建独立容器和数据库。
- 密钥类配置不支持通过命令行参数传递，避免进入进程列表和 Shell 历史。

## 3. 预期本地启动流程

代码骨架完成后，应提供以下稳定命令：

```bash
cp .env.example .env
make tools
make infra-up
make migrate-up
make seed
make run-api
```

另一个终端：

```bash
make run-worker
```

停止非数据依赖：

```bash
make infra-down
```

清空本地数据是破坏性操作，必须使用名称明确的 `make dev-reset`，执行前打印目标 Compose project/数据库并要求确认；CI 可用显式的非交互变量跳过确认。

## 4. Make 任务契约

| 命令 | 预期行为 |
| --- | --- |
| `make help` | 列出公开任务和说明 |
| `make fmt` | 格式化 Go、YAML、Markdown（不隐藏改动） |
| `make lint` | go vet、golangci-lint、配置校验 |
| `make test` | 单元测试与 race-safe 快速测试 |
| `make test-integration` | 通过 testcontainers 启动真实依赖 |
| `make test-e2e` | 启动完整本地栈验证业务场景 |
| `make build` | 生成 API/Worker 二进制 |
| `make image` | 构建最小非 root 容器镜像 |
| `make migrate-up` | 将开发数据库迁移到最新版本 |
| `make openapi-check` | 校验契约并检测破坏性变更 |
| `make kind-up` / `kind-down` | 创建/删除专用 kind 集群 |
| `make loadtest-smoke` | 小规模、可在开发机运行的 k6 测试 |

`make test` 不应要求预先安装 PostgreSQL/Redis/Kafka。耗时或有资源要求的测试使用独立任务和 build tag。

## 5. 测试策略

### 5.1 单元测试

- 使用表驱动测试、子测试和明确的 Given/When/Then 命名。
- 重点覆盖领域状态机、错误映射、幂等判断、重试边界。
- 只 mock 进程边界；不要 mock 被测模块内部实现细节。
- 对并发代码运行 `go test -race ./...`，CI 至少在主分支或定时执行。

### 5.2 集成测试

- testcontainers-go 启动固定镜像版本的 PostgreSQL、Redis 和 Kafka。
- 每个测试套件拥有隔离 schema/topic，测试完成后可靠清理。
- 必测数据库唯一约束、事务回滚、Outbox 锁竞争、消费者重复消息。

### 5.3 E2E 与负载测试

- E2E 从 HTTP 入口验证预约、领取、事件落库和查询结果。
- k6 脚本区分 smoke、baseline、stress、spike、soak。
- 负载结果保存在 `docs/benchmarks/<date>-<commit>.md`，包含环境、命令、数据集和原始摘要。
- 预期库存耗尽、重复请求与 429 不计为系统错误，但必须单独统计。

## 6. kind/Kubernetes 开发

预期流程：

```bash
make kind-up
make image
make kind-load
helm upgrade --install fanpulse deployments/helm/fanpulse \
  --namespace fanpulse --create-namespace \
  -f deployments/helm/fanpulse/values-kind.yaml
kubectl -n fanpulse get pods
```

验收清单：

- API 和 Worker 均以非 root 运行。
- readiness、liveness、startup probes 行为符合架构文档。
- 删除 API Pod 时请求可继续，终止日志显示完整排空流程。
- HPA 可通过合成负载扩容，且数据库连接预算未超限。
- `/metrics` 仅供集群内抓取。

## 7. 调试手册

### API 返回 503

1. 查看 `/health/ready` 的安全检查摘要。
2. 检查 PostgreSQL 连接、连接池等待和 Redis/Kafka 是否为该路由关键依赖。
3. 使用 request ID 关联日志与 trace；不要只重启隐藏原因。

### 奖励库存不一致

1. 立即停止对应奖励的快速路径或切到数据库基线。
2. 对比 PostgreSQL 的 `remaining_quantity`、成功 claim 数和 Redis stock。
3. 检查 pending claim、Outbox backlog、Worker/DLQ。
4. 运行只读核对，确认修复计划后再执行可审计修正。

### Kafka lag 增长

1. 区分生产突增、消费者错误、partition 倾斜和下游变慢。
2. 查看处理耗时、重试/DLQ、数据库连接等待和消息 key 分布。
3. 扩 Worker 前验证 partition 数与下游容量，避免无效扩容。

### Pod Pending / CrashLoopBackOff

- Pending：检查资源、调度约束、PVC、taint/toleration 和事件。
- CrashLoop：检查当前与 previous logs、Pod events、OOM、配置、Secret 和探针。

## 8. CI 质量门禁

PR 必须通过：格式、lint、单元测试、OpenAPI 校验、migration 检查、镜像构建和安全扫描。集成测试按资源成本选择每个 PR 或受影响路径触发；主分支必须完整执行。

发布候选还需：

- 生成 SBOM 并扫描镜像。
- 使用不可变 commit SHA tag，禁止只依赖 `latest`。
- Helm template/lint 与 kind smoke test 通过。
- changelog 和 migration/rollback 说明完备。

## 9. 常见开发节奏

单个小迭代建议控制在 1～3 天：定义 Issue 验收标准 → 建短分支 → 先写失败测试/契约 → 实现 → 本地验证 → 更新文档 → 自审 PR → squash merge。每个里程碑结束后做一次真实演示和简短复盘，把风险与后续工作写回 backlog。
