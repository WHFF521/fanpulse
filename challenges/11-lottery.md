# 第 11 关：异步抽选与可重放结果

## 业务背景

大量报名者开奖不应阻塞 HTTP 请求。管理员创建任务后由 Worker 分批处理。重试必须得到同一结果，否则失败恢复会改变中奖者。

## 第一部分：抽样算法

文件：`internal/lottery/draw.go`

### `Draw(candidates, winnerCount, seed)`

- winnerCount 不能为负或大于候选人数；0 合法并返回空 slice。
- 候选 ID 必须非空且无重复。
- 复制输入，不能修改调用方 slice。
- 使用本地 `rand.New(rand.NewSource(seed))`，不要改全局 RNG。
- Fisher-Yates shuffle 后取前 N；同样输入和 seed 必须产生同样结果。

这只是工程演示的可重放伪随机方案，不宣称满足受监管抽奖公平性。

## 第二部分：任务状态机

依据数据模型文档创建 `lotteries`、`lottery_entries`、`lottery_jobs`、`lottery_results` migration，并实现：

```go
func (s *Service) Enter(ctx context.Context, lotteryID, userID string) (Entry, bool, error)
func (s *Service) RequestDraw(ctx context.Context, lotteryID string) (Job, bool, error)
func (w *DrawWorker) Run(ctx context.Context, jobID string) error
```

要求：

- Enter 使用唯一约束幂等。
- RequestDraw 创建 PENDING 任务和 Outbox 事件，已有有效任务则返回它。
- Worker 固化候选快照和加密/受保护 seed，状态按合法路径转移。
- 结果和 COMPLETED 状态同事务提交；失败保留安全摘要。
- 公布前普通用户不能查询结果。

## 必须补充的测试

- 0 个 winner、全员中奖、重复候选、输入不变。
- 同一 seed 重试结果相同，不同 seed 通常不同（不要写概率脆弱断言）。
- Worker 在结果提交前失败可重试；提交后重复消息不新增结果。
- 中奖数不超过配置且用户唯一。

## 涉及知识

- Fisher-Yates、伪随机、可重放性。
- 异步 Job 状态机、snapshot、事务提交。
- 公平性、安全性与“演示算法”的责任边界。

## 通关测试

```powershell
go test ./internal/lottery -count=30 -v
go test -tags=integration ./tests/integration -run Lottery -v
```

## 完成标准

- 单元与状态机集成测试通过。
- 能解释为什么 `ORDER BY RANDOM() LIMIT n` 在大表和失败恢复上不理想。
