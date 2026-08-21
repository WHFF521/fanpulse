# FanPulse 编码规范与 Git 分支工作流

## 1. 总体原则

- 正确、清晰、可测试优先于抽象层数和技巧性。
- 领域规则集中在服务/领域层，HTTP、SQL、Redis、Kafka 只是适配器。
- 只为当前可验证需求设计扩展点；重复出现且边界稳定后再抽象。
- 错误、超时、取消和资源关闭是正常控制流的一部分，不作为收尾补丁。

## 2. Go 规范

### 2.1 基础规则

- 以 `gofmt`/`goimports` 为准，不手工争论格式。
- 包名短、小写、单数且表达职责；避免 `utils`、`common`、`base`。
- 导出标识符有 GoDoc；接口通常定义在使用方，保持最小方法集。
- 优先返回具体类型，只有调用方需要替换实现时才引入接口。
- 不使用 `panic` 表达可恢复业务错误；进程启动阶段不可恢复配置错误可以失败退出。

### 2.2 Context 与并发

- `context.Context` 是需要取消/超时的方法第一个参数，不存入 struct，不传 nil。
- 每个 goroutine 都必须有所有者、退出条件和可观察的错误路径。
- 使用 `errgroup.WithContext` 管理同生命周期任务；channel 由发送方关闭。
- 禁止无界 goroutine、无界 channel 和静默后台重试。
- 在最外层设置请求超时，向下传递 deadline；不要在深层任意创建更长超时。

### 2.3 错误处理

- 用 `%w` 包装错误并增加操作上下文，例如 `fmt.Errorf("claim reward: %w", err)`。
- 使用 `errors.Is/As` 判断稳定的领域错误，不匹配错误字符串。
- HTTP handler 集中把领域错误映射为 Problem Details；内部错误只向客户端返回安全摘要。
- 日志只在能采取动作的边界记录一次，避免同一错误被每层重复打印。

### 2.4 数据与事务

- 所有 SQL 参数化；查询列显式列出，禁止业务代码使用 `SELECT *`。
- 事务边界由应用用例控制；repository 不在调用者不知情时另开事务。
- 金丝雀式先查后写不能替代唯一约束或条件更新。
- 所有集合查询有确定排序和上限；分页优先 cursor，避免大 offset。
- 数据库错误转换为稳定领域错误时保留原始错误链。

### 2.5 HTTP 与事件

- Handler 只负责解析、校验、调用用例和序列化。
- 严格 JSON 解码：拒绝未知字段、限制请求体、只接受单个 JSON 值。
- 不把数据库实体直接作为 API DTO；显式映射防止意外暴露字段。
- 事件名称包含版本，事件创建后视为不可变；消费者忽略未知可选字段。
- 重试前判断操作是否幂等，所有重试有次数、退避、抖动和指标。

### 2.6 日志与指标

- 使用 `log/slog` 结构化日志；字段键保持低变动并集中定义。
- 禁止记录 JWT、密码、Secret、完整邮箱、原始幂等 key 和大 payload。
- 指标标签必须低基数；用户、请求、事件 UUID 放日志/trace，不放 label。
- 错误日志包含稳定 `error_code`，堆栈只在受控内部环境按需记录。

## 3. 项目结构约束

- `cmd/<name>/main.go` 只做依赖装配、生命周期和退出码控制。
- `internal/<domain>` 不能导入其他模块的 PostgreSQL 实现。
- `internal/platform` 不包含业务规则。
- `pkg` 默认保持为空；新增包需要在 PR 中说明仓库外复用场景。
- 生成代码放入明确目录并带生成标记；生成命令必须可复现。
- 测试文件贴近被测代码；跨模块 E2E 放 `tests/e2e`。

推荐依赖方向：

```text
HTTP/Kafka adapter -> application service -> domain
DB/Redis adapter --------------------------^ (implements ports)
```

domain 不依赖 HTTP、SQL driver、Redis client、Kafka client 或 Kubernetes SDK。

## 4. 测试规范

- 测试名说明条件与结果，例如 `TestRewardService_Claim_ReturnsExistingClaimOnRetry`。
- 并发正确性测试断言不变量：成功数、唯一数、剩余库存，而不只断言无错误。
- 时间、随机数、ID 生成通过小接口注入，使测试可重复。
- 不使用任意 `sleep` 等待异步完成；轮询明确条件并设置短超时。
- 性能 benchmark 不混入正确性单元测试；负载结论必须标注环境。
- 覆盖率是线索而非目标；核心分支必须有有意义断言。

## 5. 配置与安全规范

- 配置启动时解析一次，缺失或冲突立即失败并给出不含秘密的提示。
- Secret 不提供弱默认值；开发示例值只能用于本地。
- 依赖版本锁定，更新由 Dependabot/Renovate 或明确 PR 管理。
- 容器使用多阶段构建、固定基础镜像版本、非 root 用户和最小运行内容。
- 新增外部依赖需说明用途、维护状态、许可证、替代方案和二进制体积影响。

## 6. Git 工作流

单人项目采用轻量 GitHub Flow，不维护长期 `develop` 分支。

```text
issue/backlog -> short-lived branch -> pull request/self-review
              -> squash merge -> main -> tag/release
```

### 6.1 分支

- `main`：始终可构建，受 CI 保护，不直接提交功能代码。
- `feat/<issue>-<slug>`：功能，如 `feat/42-reward-claim`。
- `fix/<issue>-<slug>`：缺陷修复。
- `docs/<issue>-<slug>`、`chore/<issue>-<slug>`、`perf/<issue>-<slug>`。
- 分支生命周期尽量不超过 3 天；大功能用 feature flag 或可独立合并的纵向切片拆分。

### 6.2 Commit

采用 Conventional Commits：

```text
feat(reward): add atomic inventory decrement
fix(worker): skip already processed events
docs(api): define idempotency conflict response
test(reward): cover concurrent claim invariant
chore(deps): update postgres test image
```

- 一个 commit 表达一个完整意图，能独立通过基本检查。
- 标题使用祈使语气，建议不超过 72 字符；正文解释 why 和 trade-off。
- 破坏性变更使用 `!` 和 `BREAKING CHANGE:`，同时更新版本与迁移说明。
- 不在 commit 中混入无关格式化或生成文件噪声。

### 6.3 Issue 与敏捷计划

Issue 至少包含：问题/用户价值、范围、非范围、验收标准、测试思路、依赖与风险。使用标签：`type:*`、`area:*`、`priority:*`、`milestone:*`。

单人迭代使用一周节奏：

1. 选择一个可演示的迭代目标和少量 Issue。
2. 每天更新 Issue checklist，而不是额外维护重复状态文档。
3. 周末演示可运行增量，记录实际结果、偏差和下一步。
4. 未完成项重新排序，不为了“完成 sprint”降低质量门禁。

### 6.4 Pull Request 与自审

即使只有一名开发者，也通过 PR 保留设计和验证记录。PR 模板应包含：

```markdown
## Why
## What
## Design / trade-offs
## Verification
## Observability impact
## Migration / rollback
## Checklist
```

自审顺序：

1. 先读完整 diff，删除调试代码、秘密和无关改动。
2. 对照 Issue 验收标准和 API/数据模型文档。
3. 检查失败路径、并发不变量、超时、资源关闭与可观测性。
4. 本地执行受影响测试并在 PR 写入命令和结果。
5. 等 CI 全绿后 squash merge，保留清晰的主分支历史。

### 6.5 发布

- 使用语义化版本；未稳定阶段为 `v0.x.y`。
- `main` 合并不等于立即正式发布；里程碑稳定后创建带注释 tag。
- Release Notes 包含新增、修复、破坏性变更、迁移步骤、已知限制和可复现实验链接。
- 镜像同时标记版本和 commit SHA；部署清单使用不可变 digest 或 SHA tag。
- 紧急修复从最新 tag/main 建 `fix/` 分支，仍走 PR、CI 与新 patch 版本，不建立永久 hotfix 分支。

## 7. 文档与 ADR

- README 面向首次访问者；需求说明 what/why；架构说明组件与取舍；API/Data 文档是实现契约。
- 涉及数据库、消息语义、服务边界、安全或运维成本的关键决策新增 ADR。
- ADR 文件名为 `docs/adr/NNNN-short-title.md`，状态为 Proposed/Accepted/Superseded。
- 文档示例不得声称未验证的性能、可用性或完成状态。

## 8. Definition of Ready / Done

Issue 开始前应有：清晰价值、范围/非范围、可验证验收标准、关键依赖和规模合理的切片。

Issue 完成时必须：代码与测试通过；契约/迁移/文档同步；安全与可观测性已考虑；PR 自审完成；功能能在支持的本地路径演示；没有把必要修复留成无主 TODO。
