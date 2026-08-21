# FanPulse 接口定义文档

## 1. 契约原则

- 基础路径：`/api/v1`；健康检查不带版本前缀。
- 传输格式：HTTPS + JSON，字符集 UTF-8。
- 时间：RFC 3339 UTC，例如 `2026-08-21T12:00:00Z`。
- ID：UUID 字符串；服务端生成，客户端不得推断顺序。
- 字段命名：JSON 使用 `snake_case`。
- 正式机器可读契约应维护在 `api/openapi/fanpulse.yaml`，本文解释语义；实现后以 OpenAPI 契约测试防止漂移。

## 2. 通用请求规则

### 2.1 认证

```http
Authorization: Bearer <JWT>
```

活动公开读取和健康检查可匿名。用户写接口要求 `user` 或 `admin` 角色；`/admin` 接口要求 `admin`。

### 2.2 请求关联与幂等

- 客户端可传 `X-Request-ID`；缺失时服务端生成。响应始终回传该头。
- 创建关键权益的接口要求 `Idempotency-Key`，值建议为 UUID，最大 128 字符。
- 同一主体、同一路由、同一 key、同一请求摘要返回已保存的状态与响应。
- key 被不同载荷复用时返回 `409 IDEMPOTENCY_KEY_REUSED`。
- 幂等记录建议保留 24 小时；业务唯一约束永久生效。

### 2.3 分页

请求参数：`limit` 默认 20，范围 1～100；`cursor` 为不透明字符串。

```json
{
  "items": [],
  "page": {
    "next_cursor": "opaque-or-null",
    "has_more": false
  }
}
```

### 2.4 错误格式

采用 RFC 9457 风格的 Problem Details：

```json
{
  "type": "https://fanpulse.dev/problems/reward-out-of-stock",
  "title": "Reward is out of stock",
  "status": 409,
  "code": "REWARD_OUT_OF_STOCK",
  "detail": "The reward has no remaining inventory.",
  "request_id": "01K...",
  "errors": []
}
```

| HTTP | 典型 code | 语义 |
| --- | --- | --- |
| 400 | `VALIDATION_FAILED`, `IDEMPOTENCY_KEY_REQUIRED` | 输入不合法 |
| 401 | `UNAUTHENTICATED` | 未认证或 token 无效 |
| 403 | `FORBIDDEN` | 无权限或不满足资格 |
| 404 | `RESOURCE_NOT_FOUND` | 资源不存在或不可见 |
| 409 | `ALREADY_EXISTS`, `REWARD_OUT_OF_STOCK`, `IDEMPOTENCY_KEY_REUSED` | 当前状态冲突 |
| 422 | `EVENT_NOT_OPEN`, `LOTTERY_CLOSED` | 语法正确但违反业务规则 |
| 429 | `RATE_LIMITED` | 触发限流；带 `Retry-After` |
| 500 | `INTERNAL_ERROR` | 未预期错误，不泄露内部细节 |
| 503 | `DEPENDENCY_UNAVAILABLE` | 暂时无法处理或未就绪 |

## 3. 资源模型

### User

```json
{
  "id": "0198...",
  "display_name": "demo-user",
  "created_at": "2026-08-21T10:00:00Z"
}
```

### Event

```json
{
  "id": "0198...",
  "slug": "creator-anniversary-2026",
  "title": "Creator Anniversary Live 2026",
  "description": "A fictional demo event.",
  "status": "PUBLISHED",
  "registration_starts_at": "2026-08-20T00:00:00Z",
  "registration_ends_at": "2026-08-21T11:55:00Z",
  "starts_at": "2026-08-21T12:00:00Z",
  "ends_at": "2026-08-21T14:00:00Z",
  "created_at": "2026-08-01T00:00:00Z",
  "updated_at": "2026-08-01T00:00:00Z"
}
```

### RewardClaim

```json
{
  "id": "0198...",
  "reward_id": "0198...",
  "user_id": "0198...",
  "status": "SUCCEEDED",
  "claimed_at": "2026-08-21T12:00:01Z"
}
```

### LotteryJob

```json
{
  "id": "0198...",
  "lottery_id": "0198...",
  "status": "RUNNING",
  "processed_count": 5000,
  "candidate_count": 100000,
  "winner_count": 0,
  "error_summary": null,
  "created_at": "2026-08-21T15:00:00Z",
  "completed_at": null
}
```

## 4. 用户与活动接口

### `POST /api/v1/users`

创建演示用户。生产化身份注册不在 MVP。

```json
{
  "email": "demo@example.com",
  "display_name": "demo-user"
}
```

成功：`201 Created` + `User`；邮箱已存在：`409 ALREADY_EXISTS`。

### `GET /api/v1/events`

公开接口。查询参数：`status=PUBLISHED|ACTIVE|ENDED`、`limit`、`cursor`。

成功：`200 OK`，返回 `Event` 分页集合。匿名用户只能看到已发布及之后状态。

### `GET /api/v1/events/{event_id}`

成功：`200 OK` + `Event`；不可见或不存在：`404`。

### `POST /api/v1/events/{event_id}/participants`

认证：用户。请求体为空对象 `{}`；建议携带 `Idempotency-Key`，但业务唯一键本身保证重复预约安全。

首次创建：

```http
HTTP/1.1 201 Created
Location: /api/v1/events/{event_id}/participants/me
```

```json
{
  "id": "0198...",
  "event_id": "0198...",
  "user_id": "0198...",
  "joined_at": "2026-08-21T11:00:00Z"
}
```

重复请求：`200 OK` + 同一记录；窗口未开放：`422 EVENT_NOT_OPEN`。

## 5. 奖励接口

### `POST /api/v1/rewards/{reward_id}/claims`

认证：用户。必需请求头：`Idempotency-Key`。请求体为空对象 `{}`。

数据库同步基线成功：`201 Created` + `RewardClaim`。相同 key 重试返回相同状态码、资源和 `Idempotency-Replayed: true`。

Redis 异步持久化模式若启用，首次可返回：

```http
HTTP/1.1 202 Accepted
Location: /api/v1/reward-claims/{claim_id}
Retry-After: 1
```

```json
{
  "id": "0198...",
  "reward_id": "0198...",
  "user_id": "0198...",
  "status": "PENDING",
  "claimed_at": null
}
```

错误：未预约为 `403 NOT_EVENT_PARTICIPANT`；未开放为 `422 REWARD_NOT_OPEN`；库存耗尽为 `409 REWARD_OUT_OF_STOCK`。

### `GET /api/v1/reward-claims/{claim_id}`

认证：资源所有者或管理员。成功 `200`；不得通过 403/404 差异泄露其他用户资源，越权统一返回 404。

### `GET /api/v1/users/me/rewards`

认证：用户。查询参数：`limit`、`cursor`。返回当前用户领取成功或处理中权益的分页列表。

## 6. 抽选接口

### `POST /api/v1/lotteries/{lottery_id}/entries`

认证：用户。请求体 `{}`。首次 `201`，重复报名 `200` + 同一报名；截止后 `422 LOTTERY_CLOSED`。

### `POST /api/v1/admin/lotteries/{lottery_id}/draw-jobs`

认证：管理员。必需 `Idempotency-Key`。

```json
{
  "reason": "Scheduled draw after entry close"
}
```

成功：`202 Accepted` + `LotteryJob`，并通过 `Location` 指向任务查询接口。已有进行中或已完成任务时返回其状态；不同参数复用 key 返回 409。

### `GET /api/v1/lottery-jobs/{job_id}`

认证：管理员。成功：`200 OK` + `LotteryJob`。

### `GET /api/v1/users/me/lottery-results`

认证：用户。支持 `lottery_id`、`limit`、`cursor`；只返回已公布结果。

## 7. 健康与可观测性接口

### `GET /health/live`

存活返回 `200 {"status":"ok"}`。该接口不执行昂贵依赖查询。

### `GET /health/ready`

可接流量返回 `200`；排空或关键依赖不可用返回：

```json
{
  "status": "not_ready",
  "checks": {
    "postgres": "unavailable",
    "draining": "false"
  }
}
```

公开环境中不返回主机名、DSN 或错误堆栈。

### `GET /metrics`

Prometheus exposition format，仅在集群内部暴露，不通过公共 Ingress。

## 8. 事件契约

统一 envelope：

```json
{
  "event_id": "0198...",
  "event_type": "reward.claimed.v1",
  "schema_version": 1,
  "aggregate_id": "0198...",
  "occurred_at": "2026-08-21T12:00:01Z",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "data": {
    "claim_id": "0198...",
    "reward_id": "0198...",
    "user_id": "0198..."
  }
}
```

| Topic | Key | 生产事件 |
| --- | --- | --- |
| `fanpulse.event.joined.v1` | `event_id` | 活动预约完成 |
| `fanpulse.reward.claimed.v1` | `reward_id` | 奖励已持久化 |
| `fanpulse.lottery.requested.v1` | `lottery_id` | 开奖任务已创建 |
| `fanpulse.lottery.completed.v1` | `lottery_id` | 结果已提交 |
| `fanpulse.dlq.v1` | 原 topic key | 无法继续处理的事件及安全错误元数据 |

兼容规则：同一版本只新增可选字段；删除、改名、改变语义或类型必须提升事件版本。消费者忽略未知字段，并验证必需字段。

## 9. 限流与缓存语义

- 用户写接口默认按 `sub + route` 限制 10 req/s、burst 20；具体值由配置控制。
- 匿名读取按来源和路由设置更宽松的限制；代理链只信任配置的反向代理。
- 429 响应包含 `Retry-After`，限流响应不可被共享缓存。
- 活动 GET 可使用 ETag/短 TTL；用户资源与写响应设置 `Cache-Control: private, no-store`。
