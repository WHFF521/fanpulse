# 第 14 关：CI/CD、负载测试、故障实验与最终通关

## 目标

把“在我电脑上能跑”变成可复现证据：每个 PR 自动验证，镜像可追溯，部署可审查，性能数字有环境，故障行为有记录。

## 第一部分：GitHub Actions

创建 `.github/workflows/ci.yml`：

```text
checkout -> setup Go/cache -> fmt check -> go vet -> golangci-lint
         -> unit/race tests -> integration tests -> OpenAPI/Helm checks
         -> docker build -> vulnerability scan -> SBOM
```

要求：

- 权限最小化，第三方 action 固定 commit SHA。
- PR 不推正式镜像；main/tag 才发布。
- 镜像标记 commit SHA 和语义版本，不只用 latest。
- 集成服务有超时，失败上传测试和诊断产物。

可选 CD：在 `deployments/argocd/` 创建 Application；对公开演示环境使用 GitOps，凭据放 GitHub Environment/外部 secret，不写入仓库。

## 第二部分：k6 场景

创建 `loadtest/reward_claim.js`，至少包含：

- 每个虚拟用户唯一身份和 Idempotency-Key。
- 部分请求故意重放相同 key。
- `check` 区分 201/202、重复、库存耗尽、429 与非预期错误。
- thresholds 包含 P95/P99、非预期错误率。
- smoke、baseline、spike、soak 可由环境变量选择规模。

先小后大。不要在开发机直接声称 20 万并发；记录实际生成能力和瓶颈。

## 第三部分：实验报告

创建 `docs/benchmarks/template.md`，每次报告填写：日期、Git commit、硬件/OS、容器资源、Pod 数、连接池、数据量、脚本参数、QPS、P50/P95/P99、错误分类、资源曲线、瓶颈与下一实验。

创建 `docs/failure-experiments.md`，至少设计：

- 删除 API Pod。
- Redis 不可用。
- PostgreSQL 不可用/连接池耗尽。
- Kafka 不可用/consumer lag 增长。

每项写假设、步骤、预期信号、恢复条件、实际结果、改进，不在未执行前伪造结论。

## 第四部分：最终回归

```powershell
go test -race ./...
go test -tags=integration ./tests/integration/... -v
go test -tags=e2e ./tests/e2e/... -v
go test ./coursechecks -run TestLevel14DeliveryAndExperiments -v
```

再在全新 clone 或 CI runner 从 README Quick Start 开始，确保没有依赖你机器上的隐藏状态。

## 涉及知识

- CI quality gate、最小权限、软件供应链。
- GitOps、不可变制品、回滚与 roll-forward。
- load model、coordinated omission、百分位延迟。
- chaos experiment 的稳态假设与停止条件。

## 最终展示清单

- README 架构图、Quick Start 和真实完成状态。
- 一段 3～5 分钟演示：预约 → 抢奖励 → trace → Grafana → kill Pod。
- 一份真实负载报告和一份故障实验报告。
- 至少 3 个 ADR，能口述替代方案与后果。
- GitHub Actions 全绿，主分支可构建，release 使用不可变镜像标识。
- 不含真实秘密、虚假性能数据或受版权保护素材。

## 通关标准

所有测试与工程检查通过；从干净环境可启动；需求、API、数据模型与实现一致；你能独立解释幂等、并发控制、最终一致性、Kubernetes 生命周期和可观测性。此时项目才算完整，而不是仅仅“代码写完”。
