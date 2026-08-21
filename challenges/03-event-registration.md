# 第 03 关：活动预约与业务幂等

## 业务背景

活动预约会因为用户双击或网络重试重复到达。应用层应把重复预约视为同一业务结果，数据库层最终通过 `(event_id, user_id)` 唯一约束保护。

## 需要完成的代码

文件：`internal/event/domain.go`、`internal/event/service.go`

### `Event.CanJoin(now time.Time) error`

- 仅 `PUBLISHED`、`ACTIVE` 可预约。
- 窗口采用 `[RegistrationStarts, RegistrationEnds)`：开始时刻包含，结束时刻不包含。
- 非法状态或时间返回 `ErrEventNotOpen`。

### `Service.Join(ctx, eventID, userID)`

步骤：

1. 立即检查空 ID；返回稳定领域错误。
2. `events.FindByID` 获取活动，保留 repository 错误链。
3. 调用 `CanJoin(s.now())`。
4. 使用 `s.newID()` 创建候选 `Participant`。
5. 调用 `CreateOrGet`；把 `created` 原样返回。
6. 每个外部调用都传入原始 `ctx`。

不要写“先 FindParticipant 再 Insert”，因为两个并发请求都可能查不到。

## 需要补充的测试

- repository 返回错误时 `Join` 不继续写。
- `CanJoin` 在 ACTIVE 状态可用。
- context 已取消时错误不会被吞掉。
- 20 个并发 Join 最终只有一个 Participant（升级测试中的内存 repo，使其线程安全）。

## 涉及知识

- 半开时间区间、状态机。
- repository port 与依赖倒置。
- check-then-act race、数据库唯一约束、upsert。
- 业务幂等与 HTTP `Idempotency-Key` 的区别。

## 通关测试

```powershell
go test ./internal/event -v
go test -race ./internal/event -count=10
```

## 完成标准

- 当前和新增测试通过 race detector。
- 能画出两个并发预约请求在唯一约束处汇合的时序。
