# 第 05 关：PostgreSQL、迁移与 Repository

## 业务背景

PostgreSQL 是 FanPulse 的事实源。这里学习把“不能重复预约、库存不能为负”等业务不变量放到数据库约束中，而不是相信单个进程的内存判断。

## 需要完成的工程

### 1. Migration

创建：

- `migrations/000001_initial_schema.up.sql`
- `migrations/000001_initial_schema.down.sql`

先实现 `users`、`events`、`event_participants`。字段与索引参照 `docs/data-model.md`。必须有：主键、外键、非空、时间窗口 CHECK、`UNIQUE(event_id,user_id)`。

### 2. PostgreSQL 适配器

创建 `internal/storage/postgres/`，建议函数：

```go
type Store struct { /* private pool and configuration */ }
func Open(ctx context.Context, databaseURL string) (*Store, error)
func (s *Store) Pool() *pgxpool.Pool // 仅供 migration/测试装配使用
func (s *Store) Close()
func NewEventRepository(store *Store) *EventRepository
func NewParticipantRepository(store *Store) *ParticipantRepository
func NewClaimRepository(store *Store) *ClaimRepository
func (r *EventRepository) FindByID(ctx context.Context, id string) (event.Event, error)
func (r *ParticipantRepository) CreateOrGet(ctx context.Context, p event.Participant) (event.Participant, bool, error)
func (r *ClaimRepository) ClaimAtomic(ctx context.Context, rewardID, userID, claimID string) (reward.Claim, bool, error)
```

要求：

- 使用 `pgx/v5`，所有查询传 context。
- 显式列名；正确关闭 rows；错误包装带操作名。
- `CreateOrGet` 用唯一约束或 upsert 处理竞争，不写先查后插。
- 设置连接池上限并解释总连接预算。

### 3. 集成测试

仓库已在 `tests/integration/postgres_test.go` 提供第一批带 `//go:build integration` 的测试。先添加 `pgx/v5` 依赖，并设置 `FANPULSE_TEST_DATABASE_URL` 指向专用测试库。随后把环境升级为 testcontainers-go 自动启动 PostgreSQL，避免依赖手工服务。测试至少验证：迁移可执行；重复预约只有一行；外键阻止无效用户；取消 context 能中止查询。

### 4. ADR

创建 `docs/adr/0001-use-postgresql.md`，写 Context、Decision、Alternatives、Consequences 和 Accepted 状态。

## 涉及知识

- SQL DDL、约束、索引、事务隔离。
- 连接池不是“连接越多越快”。
- migration 与应用启动解耦。
- testcontainers、build tags。

## 通关测试

```powershell
go test ./coursechecks -run TestLevel05Migrations -v
go test -tags=integration ./tests/integration -run 'Postgres|Participant' -v
go test ./internal/event -v
```

## 常见错误

- down migration 删除顺序违反外键。
- 用字符串拼接 SQL。
- 把 `sql.ErrNoRows`/pgx no rows 当成 500 而非领域 not found。
- 测试连接开发者真实数据库，造成数据污染。

## 完成标准

- 静态与真实 PostgreSQL 集成测试通过。
- 能从唯一约束冲突解释 `created=false` 如何产生。
- migration 在空库上能 up、down、再次 up。
