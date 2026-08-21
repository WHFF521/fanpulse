# FanPulse

FanPulse 是一个面向大型线上粉丝活动的高并发、云原生后端平台。项目以“活动开始瞬间大量用户预约、领取限量奖励并参加抽选”为核心场景，用可运行、可测试、可观测的实现展示 Go Backend、Platform Engineering 与 SRE 能力。

> 当前状态：文档与工程设计阶段。性能数据必须来自可复现实验；项目不预设或伪造吞吐量指标。

## 学习闯关模式

仓库同时提供一条由测试驱动的完整实现路线。项目框架、函数签名和失败测试已经准备好，业务代码中的 `TODO(level-XX)` 需要学习者亲自完成。请从 [闯关手册](challenges/README.md) 的第 00 关开始，不要直接运行并试图一次修复全部测试。

```powershell
./scripts/check-level.ps1 -Level 0
./scripts/check-level.ps1 -Level 1
```

初始状态下第 00 关通过、后续关卡失败是有意设计；每完成一关，其对应测试应由红变绿。

## 项目目标

- 正确处理重复请求、限量库存和消息重复投递。
- 展示从 Go 代码、数据存储到 Kubernetes、CI/CD 和可观测性的完整工程闭环。
- 以单人可持续开发为前提，先做深关键问题，再逐步增加基础设施复杂度。
- 形成适合公开展示、面试讲解和复盘的架构决策、测试结果与故障实验记录。

## 核心能力

- 活动浏览与预约，数据库唯一约束保证不重复报名。
- 限量奖励领取，支持 `Idempotency-Key`、原子扣减和最终唯一性保护。
- 异步抽选任务，包含状态机、重试和失败处理。
- Outbox、Kafka 消费者幂等与至少一次投递语义。
- Redis 限流、幂等状态和高并发库存优化。
- Kubernetes 健康检查、优雅停机、HPA、PDB 和 NetworkPolicy。
- Prometheus 指标、结构化日志和 OpenTelemetry 链路追踪。
- 单元、集成、端到端、负载与故障测试。

## 架构摘要

第一阶段采用“模块化单体 API + 独立异步 Worker”：业务模块在代码中隔离，但共享一个 PostgreSQL 数据库。这样既能清楚展示领域边界，也适合单人快速迭代。只有在压测或运维证据表明需要独立扩缩容时，才拆分物理微服务。

```text
Client -> Ingress -> API
                       |-- PostgreSQL
                       |-- Redis
                       `-- Outbox -> Kafka -> Worker -> PostgreSQL

Metrics / Logs / Traces -> Prometheus / Grafana / Loki / Tempo
```

详细设计见 [架构设计](docs/architecture.md)。

## 技术栈

| 分类 | 选择 |
| --- | --- |
| 语言与 API | Go、REST、OpenAPI 3.1 |
| 数据 | PostgreSQL、Redis |
| 消息 | Kafka |
| 本地环境 | Docker Compose、kind |
| 平台 | Kubernetes、Helm、Argo CD |
| IaC | Terraform（本地演示与 AWS 参考架构分离） |
| 可观测性 | Prometheus、Grafana、OpenTelemetry、Loki、Tempo |
| 测试 | Go testing、testcontainers-go、k6 |
| CI/CD | GitHub Actions、容器镜像扫描、GitOps |

## 仓库规划

```text
fanpulse/
├── cmd/api/                 # HTTP API 进程
├── cmd/worker/              # Kafka/Outbox/抽选 Worker
├── internal/                # 按领域模块组织的私有代码
├── pkg/                     # 仅存放真正可复用的通用包
├── api/openapi/             # OpenAPI 契约
├── migrations/              # PostgreSQL 迁移
├── deployments/helm/        # Helm Chart
├── deployments/argocd/      # GitOps 配置
├── terraform/               # 基础设施定义
├── loadtest/                # k6 场景与结果说明
├── docs/                    # 设计、规范、ADR 与实验记录
├── docker-compose.yml
├── Makefile
└── README.md
```

## 文档导航

- [需求规格说明书](docs/requirements-specification.md)
- [架构设计文档](docs/architecture.md)
- [接口定义文档](docs/api-contract.md)
- [数据库与数据模型设计](docs/data-model.md)
- [开发环境指南](docs/development-guide.md)
- [编码规范与 Git 工作流](docs/engineering-guidelines.md)
- [项目闯关手册](challenges/README.md)

## 里程碑

1. **M1 基础后端**：活动、预约、数据库迁移、分层边界和基础测试。
2. **M2 并发正确性**：奖励领取、幂等、数据库原子扣减和 k6 基线。
3. **M3 高并发路径**：Redis、限流、库存 Lua 脚本及一致性校验。
4. **M4 事件驱动**：Outbox、Kafka、消费者幂等、重试与 DLQ。
5. **M5 云原生运行**：Docker、kind、Helm、健康检查、优雅停机、HPA/PDB。
6. **M6 生产工程**：指标、日志、追踪、CI/CD、故障实验和公开报告。

## 快速开始

代码骨架建立后，预期的本地流程如下：

```bash
cp .env.example .env
docker compose up -d postgres redis kafka
make migrate-up
make run-api
make run-worker
```

验证服务：

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

当前仓库尚处于设计阶段，以上命令是需要在对应里程碑中兑现的公开契约，而不是已完成声明。完整说明见 [开发环境指南](docs/development-guide.md)。

## 质量原则

- 正确性优先于峰值数字；库存不得超发，重复请求不得产生重复权益。
- 所有公开性能结论都记录提交版本、硬件、拓扑、数据量和 k6 脚本。
- 关键业务操作必须可追踪，日志不得泄露令牌、密码或完整个人信息。
- 设计变化通过 ADR 记录，不为“看起来高级”而引入组件。

## 开源与免责声明

FanPulse 是独立的学习与作品集项目，不隶属于、也不复制任何现实艺人团体或平台。示例人物、活动和数据均应使用虚构内容。许可证将在首个可运行版本发布前确定，暂定优先考虑 Apache-2.0。
