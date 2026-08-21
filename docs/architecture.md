# FanPulse 架构设计文档

## 1. 架构目标

FanPulse 的架构首先保证奖励不超发、请求与消费可幂等，其次保证系统可扩展、可观测、可测试。作为单人作品集项目，架构还必须允许分阶段交付，不能要求所有外部组件同时完成才能运行。

## 2. 关键决策摘要

| 决策 | 选择 | 原因 |
| --- | --- | --- |
| 应用形态 | 模块化单体 API + 独立 Worker | 降低单人开发与部署成本，同时保留领域边界 |
| API | REST + OpenAPI 3.1 | 易演示、易生成客户端与契约测试 |
| 事实源 | PostgreSQL | 事务、约束和查询能力适合核心业务 |
| 高并发辅助 | Redis | 原子 Lua、限流、短期幂等状态与缓存 |
| 消息 | Kafka | 展示事件驱动、至少一次投递与消费延迟治理 |
| 写入一致性 | Transactional Outbox | 避免数据库成功但事件丢失 |
| 部署 | Docker Compose -> kind/Helm -> 云参考 | 本地可复现，按证据演进 |
| 可观测性 | OTel + Prometheus/Loki/Tempo/Grafana | 统一 metrics、logs、traces 关联 |

这些选择应分别通过 ADR 固化；如果实现阶段改变，先更新 ADR，再同步本文。

## 3. 系统上下文

```text
                 +------------------+
Visitor/User --->|                  |
Admin ---------->|     FanPulse     |----> Observability stack
                 |                  |
                 +--------+---------+
                          |
                          +--------------> PostgreSQL
                          +--------------> Redis
                          `--------------> Kafka
```

外部身份提供商、真实通知平台和支付服务均不在 MVP。认证通过本地 JWT issuer 模拟，通知通过事件与日志适配器演示。

## 4. 容器与运行时视图

```text
                       Kubernetes Ingress
                               |
                         +-----v-----+
                         | API Pods  |
                         +--+--+--+--+
                            |  |  |
              +-------------+  |  +-------------+
              |                |                |
        +-----v------+   +-----v------+   +-----v-----+
        | PostgreSQL |   |   Redis    |   |  Kafka    |
        +-----+------+   +------------+   +-----+-----+
              ^                                  |
              |                            +-----v-----+
              +----------------------------+ Worker   |
                                           +-----------+

API/Worker --OTLP--> OpenTelemetry Collector
Collector ----------> Prometheus / Loki / Tempo --> Grafana
```

本地 Docker Compose 可直接运行依赖与进程；kind 环境用于验证探针、滚动发布、HPA、PDB 和网络策略。数据库、Redis、Kafka 在云参考架构中优先使用托管服务，不把有状态组件运维能力与应用能力混为一谈。

## 5. 应用模块

```text
internal/
├── identity/      # 用户与认证主体
├── event/         # 活动及预约
├── reward/        # 奖励库存与领取
├── lottery/       # 报名、开奖任务与结果
├── messaging/     # Outbox、事件封装、消费幂等
├── platform/      # 配置、HTTP、日志、追踪、健康检查
└── testkit/       # 仅供测试的 builder/helper
```

模块内部建议按职责拆分 `domain.go`、`service.go`、`repository.go`、`handler.go`。领域服务依赖本模块定义的小接口，PostgreSQL、Redis 和 Kafka 适配器在外层实现。禁止业务模块通过数据库表直接写入其他模块；跨模块调用经应用服务或领域事件完成。

`pkg/` 只放仓库外也有明确复用价值的稳定包。普通内部工具留在 `internal/platform`，避免过早制造公共 API。

## 6. 关键流程

### 6.1 活动预约

```text
Client -> API -> validate auth/window
              -> INSERT event_participants
                 ON CONFLICT(event_id,user_id) return existing
              -> commit participant + outbox event
              -> response
```

数据库唯一约束是并发正确性的最终保障。重复预约在业务上视为成功读取，而不是 500。

### 6.2 奖励领取：M2 数据库基线

单个事务内执行：

1. 尝试登记幂等请求或读取已有结果。
2. 原子执行 `UPDATE rewards SET remaining_quantity = remaining_quantity - 1 WHERE id = $1 AND remaining_quantity > 0`。
3. 插入 `reward_claims`，唯一键为 `(reward_id, user_id)`。
4. 插入 Outbox 事件。
5. 提交并保存幂等响应。

发生唯一键竞争时读取既有领取记录；更新影响行为 0 时返回库存耗尽。数据库路径先作为正确性基线与降级路径。

### 6.3 奖励领取：M3 Redis 快速路径

```text
Client -> API -> Idempotency state
              -> Redis Lua: user dedupe + stock check + decrement
              -> durable command/outbox
              -> response ACCEPTED/SUCCEEDED
                           |
                           v
                    Worker persists claim
```

Lua 脚本保证 Redis 内检查与扣减原子化；数据库唯一约束继续阻止重复权益。实现前必须明确返回语义：若持久化是异步的，接口使用 `202 PENDING`，不得在持久化前宣称最终成功。Redis 与数据库通过核对任务修复偏差。

### 6.4 Outbox 发布

```text
Business transaction
├── domain row(s)
└── outbox_events(status=PENDING)
         |
         v
Publisher: SELECT ... FOR UPDATE SKIP LOCKED
         -> publish Kafka
         -> mark PUBLISHED
```

发布成功但状态更新前崩溃会导致重复发布，这是允许的；消费者必须幂等。Outbox 发布器使用批次、租约/锁、指数退避和最大尝试次数，避免毒消息阻塞整个队列。

### 6.5 Kafka 消费幂等

```text
receive event
  -> begin DB transaction
  -> INSERT processed_events(event_id, consumer_name)
  -> apply business side effect
  -> commit DB
  -> commit Kafka offset
```

若唯一键已存在，则跳过副作用并提交 offset。不可重试错误进入 DLQ；可重试错误不提前提交 offset。事件 schema 包含 `event_id`、`event_type`、`occurred_at`、`aggregate_id`、`schema_version`、`trace_id` 和 `data`。

### 6.6 抽选

管理员请求仅创建任务并返回 202。Worker 对候选人建立确定性快照，使用可记录的随机种子进行无放回抽样，在同一抽选内保证中奖用户唯一。重试复用快照与种子，避免生成不同结果。面向真实公平性要求时需引入经过审计的随机方案；MVP 明确仅作工程演示。

## 7. 一致性模型

| 数据 | 模型 | 说明 |
| --- | --- | --- |
| 奖励库存与领取记录 | 单库事务强一致（基线） | 原子扣减与唯一约束 |
| Redis 库存 | 最终一致 | 可从 PostgreSQL 重建并周期核对 |
| 领域事件发布 | 最终一致 | Outbox 至少一次发布 |
| 消费者副作用 | 幂等最终一致 | `processed_events` 防重复 |
| 活动列表缓存 | 最终一致 | TTL + 写后失效 |
| 抽选任务状态与结果 | 数据库事务一致 | 任务完成与结果提交同事务 |

## 8. 可扩展性与过载保护

- API 无状态化，按 CPU 和业务指标水平扩容。
- 读多的活动数据采用 Cache-Aside；TTL 加随机抖动，热点键使用 singleflight/互斥重建。
- 用户级和全局令牌桶在入口早期拒绝过载，返回 429。
- 数据库连接池设置硬上限；总连接预算按最大 Pod 数反推，避免 HPA 放大数据库故障。
- Worker 并发受 Kafka partition 数、数据库连接和下游容量共同限制。
- 超时必须逐层递减；仅对幂等且临时性的失败执行有界重试并加入抖动。

## 9. 可靠性与 Kubernetes 生命周期

- `/health/live` 只检查进程是否卡死，不因 PostgreSQL/Kafka 短暂故障重启进程。
- `/health/ready` 检查服务是否完成启动、未处于排空状态及关键同步依赖是否可用。
- 使用 `startupProbe` 保护慢启动，避免 liveness 过早介入。
- SIGTERM 后立即标记 not ready，等待传播窗口，关闭监听，等待在途请求和 Worker 当前批次，再关闭客户端连接。
- API 配置最少 2 副本、RollingUpdate、PDB 和 topology spread；本地演示可降低副本数。
- NetworkPolicy 仅允许必要的入口、DNS 和数据依赖流量。

## 10. 可观测性

### 10.1 指标

- HTTP：请求量、状态码、持续时间直方图、在途请求。
- 奖励：领取成功、重复、库存耗尽、幂等冲突、库存核对偏差。
- Kafka：消费量、失败、重试、DLQ、consumer lag。
- 数据库：查询延迟、错误、连接池使用/等待。
- Go runtime：goroutine、GC、堆内存、CPU。

标签禁止放 `user_id`、`event_id`、原始路径等高基数值；HTTP 路径使用路由模板。

### 10.2 日志与追踪

日志字段至少包含 `timestamp`、`level`、`service`、`message`、`request_id`、`trace_id`、`error_code`。错误保留包装链，边界层集中决定 HTTP 状态和安全消息。

HTTP、Redis、PostgreSQL、Kafka publish/consume 建立 span；异步事件传播 W3C Trace Context。采样率通过环境配置，高错误率时保留错误 trace。

### 10.3 初始 SLO（演示环境）

- API 可用性目标：30 天 99.5%，排除计划内本地停机。
- 已接受请求正确性：库存超发为 0，重复权益为 0。
- 告警从用户症状出发：5xx、P99、库存偏差、消费延迟和 DLQ 增长。

## 11. 安全设计

- JWT 校验 issuer、audience、expiry 与签名；用户 ID 从 token 获取，不信任请求体中的身份。
- 管理 API 使用角色声明，并对关键变更写审计日志。
- 使用参数化查询、严格 JSON 解码、字段长度/枚举校验和请求体上限。
- CORS 默认拒绝，按演示前端来源显式放行。
- Secret 不进入镜像、Git、日志或 trace；本地使用 `.env`，集群通过 Secret 引用。
- 容器以非 root、只读根文件系统和最小 Linux capabilities 运行。

## 12. 部署拓扑与演进

| 阶段 | 拓扑 | 退出条件 |
| --- | --- | --- |
| M1-M2 | Go 进程 + Docker Compose 依赖 | 核心并发正确性测试通过 |
| M3-M4 | API + Worker + PostgreSQL/Redis/Kafka | Outbox、消费幂等和恢复测试通过 |
| M5 | kind + Helm + 可观测性 | 探针、扩缩容、滚动与故障实验通过 |
| M6 | AWS 参考 Terraform | 计划可审查、成本和安全假设已记录；不要求长期运行 |

只有满足至少一个条件才拆分 API 模块为独立服务：需要独立扩缩容；发布节奏或故障域必须隔离；数据所有权已经清晰；压测证实模块边界带来收益。

## 13. 已知取舍

- Kafka 增加本地资源占用，但能真实展示投递语义和消费治理；开发模式保持单 broker。
- Redis 快速路径提升吞吐但增加核对复杂度，因此必须先完成数据库基线。
- 单库降低分布式事务难度，但模块边界靠代码审查维持；这是单人项目的有意选择。
- 本地 SLO 不是商业承诺，只用于驱动指标、告警和实验设计。
