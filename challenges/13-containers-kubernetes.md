# 第 13 关：Docker、Compose、Helm 与 Kubernetes 可靠性

## 业务背景

本关把已完成的程序放进一致、最小、安全的容器，并在 kind 中验证生命周期与水平扩展。不是简单写几份 YAML。

## 第一部分：Dockerfile

创建多阶段 `Dockerfile`：

- builder 固定 Go 基础镜像版本，利用 module cache。
- 分别构建 API/Worker，建议通过 build arg 选择 target 或使用两个最终 stage。
- 最终镜像固定 digest/明确版本，包含 CA 和时区需求但不带编译器。
- 非 root `USER`、只读友好、明确 `ENTRYPOINT`。
- 使用 `.dockerignore` 排除 `.git`、缓存、文档大文件和本地秘密。

## 第二部分：Docker Compose

创建 `docker-compose.yml`，包含 PostgreSQL、Redis、Kafka、API、Worker。每个有状态依赖有 healthcheck，应用通过 service name 连接并等待健康而非固定 sleep。数据卷名称带项目作用域。

不要默认把数据库/Kafka 管理端口暴露到所有网卡；本地凭据只能用于 development。

## 第三部分：Helm Chart

在 `deployments/helm/fanpulse/` 创建：

- `Chart.yaml`、`values.yaml`、`values-kind.yaml`。
- API/Worker Deployment 与 Service。
- Ingress、ConfigMap、Secret 引用、ServiceAccount。
- readiness/liveness/startup probes。
- requests/limits、RollingUpdate、PDB、HPA、NetworkPolicy。

连接池上限必须结合 HPA max replicas 计算。例如数据库预算 80、API 最大 10 Pod、Worker 最大 5 Pod时，不能每 Pod 默认开 20 个连接。

## 第四部分：kind 验证

执行并记录：

1. Helm lint/template。
2. 部署到专用 kind cluster。
3. 连续请求时删除一个 API Pod，观察可用性与优雅停机日志。
4. 触发 CPU/业务指标负载，观察 HPA。
5. 检查 PDB 和 NetworkPolicy 是否真的产生预期限制。

## 涉及知识

- 容器层缓存、PID 1、信号传播、非 root。
- K8s controller、probe、Deployment rollout。
- HPA 与下游容量、PDB 与节点维护。
- Secret 对象不是完整秘密管理方案。

## 通关测试

```powershell
go test ./coursechecks -run TestLevel13ContainersAndKubernetes -v
docker compose config
helm lint deployments/helm/fanpulse
helm template fanpulse deployments/helm/fanpulse --values deployments/helm/fanpulse/values-kind.yaml
```

随后运行你写的 kind smoke/E2E 脚本；脚本必须设超时并在失败时保留诊断信息。

## 完成标准

- 静态、Compose、Helm 和 kind 测试通过。
- 删除 API Pod 不导致明显非预期 5xx；证据写入实验记录。
- 能从 `kubectl apply` 解释 API Server 到 kubelet 的控制链路。
