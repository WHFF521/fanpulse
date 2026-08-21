# 第 06 关：奖励库存与并发正确性

## 业务背景

100 份奖励面对 1,000 个并发用户时，成功领取必须恰好不超过 100，库存不能为负，同一用户不能领取两次。这是项目最重要的不变量。

## 第一部分：完成内存模型

文件：`internal/reward/reward.go`

### `NewInventory(quantity int64) *Inventory`

初始化库存与 `claimed` map。建议修改签名为返回 error 以拒绝负数；如果修改，需要同步更新测试和调用方，并在 PR 解释 API 变化。

### `Inventory.Claim(userID string) error`

在同一个临界区内执行：检查重复 → 检查库存 → 记录用户 → 扣减。不能把锁拆开，也不要用 `RWMutex` 假装复合写操作是读。

### `Inventory.Remaining() int64`

并发安全地读取剩余库存。

## 第二部分：实现 PostgreSQL 原子领取

扩展 migration 创建 `rewards`、`reward_claims`、`outbox_events`（Outbox 可在第 09 关启用）。实现：

```go
func (r *ClaimRepository) ClaimAtomic(
    ctx context.Context, rewardID, userID, claimID string,
) (reward.Claim, bool, error)
```

事务内条件扣减再插入 claim。重复用户导致插入冲突时整个扣减必须回滚，并返回已有 claim。库存更新影响 0 行时区分“不存在/未开放/库存耗尽”。

## 必须补充的集成测试

- 库存 10、100 个独立用户并发请求，成功恰好 10。
- 同一用户 20 个并发请求，只出现一个 claim，库存只减 1。
- 人为让 claim 插入失败，库存回滚。

## 涉及知识

- mutex、critical section、race detector。
- lost update、条件 UPDATE、唯一约束。
- 事务原子性和错误后的 rollback。
- 为什么“加 Pod”不能修复并发正确性。

## 通关测试

```powershell
go test -race ./internal/reward -count=20
go test -tags=integration ./tests/integration -run Reward -count=3 -v
```

## 完成标准

- race detector 无告警。
- 任意测试重复次数都没有超发和重复权益。
- 能解释内存锁为何不能跨 Pod，以及数据库方案为何能跨 Pod。
