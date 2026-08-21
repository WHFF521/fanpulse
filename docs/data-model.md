# FanPulse 数据库与数据模型设计

## 1. 设计原则

- PostgreSQL 是业务事实源，Redis 是可重建的加速与协调层。
- 约束由数据库表达，不能只依赖应用层先查后写。
- 所有时间使用 `timestamptz` 并以 UTC 读写。
- 主键使用 UUID；实现优先 UUIDv7 以改善索引局部性。
- 金额当前不在范围；若以后加入，使用最小货币单位整数而非浮点数。
- 迁移只前进、可审计；破坏性变更采用 expand/migrate/contract。

## 2. 关系概览

```text
users 1---* event_participants *---1 events
  |                                      |
  |                                      +---* rewards 1---* reward_claims *---1 users
  |                                      |
  +---* lottery_entries *---1 lotteries *---1 events
  |                              |
  +---* lottery_results *---------+

lotteries 1---* lottery_jobs
outbox_events              # 事务事件发布
processed_events           # 消费者幂等
```

## 3. 枚举与状态机

建议在数据库中使用受 CHECK 约束的 `text`，便于演进；Go 中定义强类型常量。

| 实体 | 状态 | 合法转移 |
| --- | --- | --- |
| events | `DRAFT`, `PUBLISHED`, `ACTIVE`, `ENDED`, `CANCELLED` | DRAFT→PUBLISHED/CANCELLED；PUBLISHED→ACTIVE/CANCELLED；ACTIVE→ENDED/CANCELLED |
| rewards | `DRAFT`, `ACTIVE`, `EXHAUSTED`, `CLOSED` | DRAFT→ACTIVE；ACTIVE→EXHAUSTED/CLOSED |
| reward_claims | `PENDING`, `SUCCEEDED`, `FAILED` | PENDING→SUCCEEDED/FAILED |
| lotteries | `DRAFT`, `OPEN`, `CLOSED`, `DRAWING`, `COMPLETED`, `FAILED` | 顺序推进，FAILED 可人工重开新任务 |
| lottery_jobs | `PENDING`, `RUNNING`, `COMPLETED`, `FAILED` | PENDING→RUNNING；RUNNING→COMPLETED/FAILED |
| outbox_events | `PENDING`, `PUBLISHED`, `FAILED` | PENDING→PUBLISHED/FAILED；FAILED 可运维重置为 PENDING |

## 4. 表定义

以下是逻辑定义；首个实现应生成版本化 SQL migration，并用集成测试验证约束。

### 4.1 `users`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| email | text | NOT NULL；保存规范化小写值 |
| display_name | varchar(80) | NOT NULL |
| role | text | NOT NULL，CHECK `USER/ADMIN` |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |

索引：`UNIQUE (email)`。公开 API 不返回 email。

### 4.2 `events`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| slug | varchar(120) | NOT NULL，URL 稳定标识 |
| title | varchar(200) | NOT NULL |
| description | text | NOT NULL DEFAULT '' |
| status | text | NOT NULL + CHECK |
| registration_starts_at | timestamptz | NOT NULL |
| registration_ends_at | timestamptz | NOT NULL |
| starts_at | timestamptz | NOT NULL |
| ends_at | timestamptz | NOT NULL |
| created_at / updated_at | timestamptz | NOT NULL |

约束：`UNIQUE(slug)`；所有时间窗口起点必须早于终点。索引：`(status, starts_at, id)` 支持游标分页。

### 4.3 `event_participants`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| event_id | uuid | FK events，NOT NULL |
| user_id | uuid | FK users，NOT NULL |
| joined_at | timestamptz | NOT NULL |

约束：`UNIQUE(event_id, user_id)`。索引：`(user_id, joined_at DESC, id)`。

### 4.4 `rewards`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| event_id | uuid | FK events，NOT NULL |
| name | varchar(160) | NOT NULL |
| status | text | NOT NULL + CHECK |
| total_quantity | bigint | NOT NULL，>= 0 |
| remaining_quantity | bigint | NOT NULL，0..total_quantity |
| claim_starts_at / claim_ends_at | timestamptz | NOT NULL |
| created_at / updated_at | timestamptz | NOT NULL |

索引：`(event_id, status)`。库存扣减只允许条件更新，禁止“先 SELECT 再 UPDATE”。

### 4.5 `reward_claims`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| reward_id | uuid | FK rewards，NOT NULL |
| user_id | uuid | FK users，NOT NULL |
| status | text | NOT NULL + CHECK |
| idempotency_key_hash | char(64) | 可空；只保存带服务端盐的摘要 |
| failure_code | varchar(80) | 可空，安全业务码 |
| claimed_at | timestamptz | 成功时非空 |
| created_at / updated_at | timestamptz | NOT NULL |

约束：`UNIQUE(reward_id, user_id)`；可添加 `UNIQUE(user_id, idempotency_key_hash) WHERE idempotency_key_hash IS NOT NULL`。索引：`(user_id, created_at DESC, id)`。

### 4.6 `lotteries`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| event_id | uuid | FK events，NOT NULL |
| name | varchar(160) | NOT NULL |
| status | text | NOT NULL + CHECK |
| winner_quantity | integer | NOT NULL，> 0 |
| entry_starts_at / entry_ends_at | timestamptz | NOT NULL |
| result_published_at | timestamptz | 可空 |
| created_at / updated_at | timestamptz | NOT NULL |

索引：`(event_id, status)`。

### 4.7 `lottery_entries`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| lottery_id | uuid | FK lotteries，NOT NULL |
| user_id | uuid | FK users，NOT NULL |
| entered_at | timestamptz | NOT NULL |

约束：`UNIQUE(lottery_id, user_id)`。索引：`(lottery_id, id)` 支持稳定批处理。

### 4.8 `lottery_jobs`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| lottery_id | uuid | FK lotteries，NOT NULL |
| status | text | NOT NULL + CHECK |
| random_seed_ciphertext | bytea | NOT NULL；结果发布前不可公开 |
| candidate_count | bigint | NOT NULL DEFAULT 0 |
| processed_count | bigint | NOT NULL DEFAULT 0 |
| winner_count | integer | NOT NULL DEFAULT 0 |
| attempt_count | integer | NOT NULL DEFAULT 0 |
| error_summary | text | 可空，不存堆栈/敏感数据 |
| started_at / completed_at | timestamptz | 可空 |
| created_at / updated_at | timestamptz | NOT NULL |

约束：每个 lottery 最多一个非失败的有效开奖任务，可用部分唯一索引实现。

### 4.9 `lottery_results`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK |
| lottery_id | uuid | FK lotteries，NOT NULL |
| job_id | uuid | FK lottery_jobs，NOT NULL |
| user_id | uuid | FK users，NOT NULL |
| rank | integer | 可空；需要排序时使用 |
| created_at | timestamptz | NOT NULL |

约束：`UNIQUE(lottery_id, user_id)`、`UNIQUE(job_id, rank)`（rank 非空时）。

### 4.10 `outbox_events`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| id | uuid | PK，同时作为 event_id |
| aggregate_type | varchar(80) | NOT NULL |
| aggregate_id | uuid | NOT NULL |
| event_type | varchar(120) | NOT NULL，含版本 |
| payload | jsonb | NOT NULL |
| trace_parent | varchar(255) | 可空 |
| status | text | NOT NULL DEFAULT PENDING |
| attempt_count | integer | NOT NULL DEFAULT 0 |
| available_at | timestamptz | NOT NULL |
| published_at | timestamptz | 可空 |
| last_error | text | 可空，截断且脱敏 |
| created_at | timestamptz | NOT NULL |

索引：`(status, available_at, created_at)`；发布批次用 `FOR UPDATE SKIP LOCKED`。

### 4.11 `processed_events`

| 列 | 类型 | 约束/说明 |
| --- | --- | --- |
| event_id | uuid | 复合 PK |
| consumer_name | varchar(120) | 复合 PK |
| event_type | varchar(120) | NOT NULL |
| processed_at | timestamptz | NOT NULL |

主键：`(event_id, consumer_name)`，允许不同消费者各处理一次同一事件。按保留政策归档/清理前，必须确保 Kafka 不会重放到该时间点之前。

## 5. 关键事务伪代码

### 5.1 原子奖励领取

```sql
BEGIN;

UPDATE rewards
SET remaining_quantity = remaining_quantity - 1,
    updated_at = now()
WHERE id = $1
  AND status = 'ACTIVE'
  AND remaining_quantity > 0
RETURNING id;

-- 影响 0 行：映射为未开放或库存耗尽。
-- 成功后插入领取；唯一冲突时回滚，并读取已有领取。

INSERT INTO reward_claims (...)
VALUES (...);

INSERT INTO outbox_events (...)
VALUES (...);

COMMIT;
```

如果插入领取因重复用户失败，整个事务回滚，之前的扣减也回滚，因此不会少库存。

### 5.2 消费者幂等事务

```sql
BEGIN;

INSERT INTO processed_events(event_id, consumer_name, event_type, processed_at)
VALUES ($1, $2, $3, now())
ON CONFLICT DO NOTHING;

-- 仅当插入成功时执行业务副作用。

COMMIT;
```

应用必须检查 affected rows，不能在冲突后仍执行副作用。

## 6. Redis 数据模型

Redis key 必须带环境前缀，例如 `fanpulse:dev:`，避免共享实例冲突。

| Key 模式 | 类型 | 用途 | TTL/恢复 |
| --- | --- | --- | --- |
| `fanpulse:{env}:idem:{user}:{route}:{hash}` | Hash/String | PROCESSING、状态码、资源 ID、响应摘要 | 24h；DB 唯一键兜底 |
| `fanpulse:{env}:reward:{reward}:stock` | String | 可领取库存 | 到领取结束 + 缓冲；从 DB 重建 |
| `fanpulse:{env}:reward:{reward}:users` | Set/分片结构 | Redis 快速去重 | 同库存 key；从 claims 重建 |
| `fanpulse:{env}:ratelimit:{subject}:{route}` | Hash/String | Token bucket 状态 | 窗口后自动过期 |
| `fanpulse:{env}:event:{event}` | String(JSON) | 活动详情缓存 | 30～120s 抖动 TTL |

Redis Cluster 环境中 Lua 涉及的 key 必须共享 hash tag，例如 `{reward_id}`，保证位于同一 slot。不得把完整邮箱、JWT 或原始幂等 key 放入 key 名。

## 7. 删除、保留与审计

- MVP 对核心业务记录不做物理删除；状态变更保留审计时间。
- 演示用户的删除需求采用匿名化：移除 email、替换 display name，同时保留不可识别的业务完整性。
- Outbox payload 和日志不得承载不必要个人信息。
- 具体保留期在公开部署前配置化并记录；本地种子数据可由明确的 reset 命令重建。

## 8. 迁移规范

- 文件名：`000001_create_users.up.sql` / `.down.sql`，或由选定工具生成等价版本。
- migration 在应用启动外独立执行；多个 Pod 不并发自动迁移。
- 新列先允许空/提供默认值，回填后再收紧；索引在线创建策略在生产参考中单独说明。
- 每个迁移在空库与上一版本快照上测试；down 不安全时提供前向修复说明而非伪造可回滚性。
- 禁止修改已进入主分支并发布的 migration；追加新 migration 修正。
