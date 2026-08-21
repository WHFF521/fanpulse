# 第 01 关：配置解析与错误语义

## 业务背景

API 和 Worker 必须从环境读取地址、数据库连接和停机宽限期。错误配置应该在启动时立即暴露，而不是在流量进入后随机失败。

## 需要完成的代码

文件：`internal/platform/config/config.go`

### `LoadFrom(getenv func(string) string) (Config, error)`

作用：通过注入的 `getenv` 读取配置，返回不可变的配置快照。

要求：

- `FANPULSE_ENV` 默认 `development`。
- `FANPULSE_HTTP_ADDR` 默认 `:8080`。
- `FANPULSE_SHUTDOWN_TIMEOUT` 默认 `15s`，用 `time.ParseDuration` 解析。
- `FANPULSE_DATABASE_URL` 必填，空值返回包装后的 `ErrInvalidConfig`。
- 非法 duration 返回包装后的 `ErrInvalidConfig`，错误文本说明是哪个字段，但不得输出数据库 URL。

建议你再补两个测试：显式配置覆盖默认值；负数/零停机时间被拒绝。

## 涉及知识

- 纯函数与依赖注入。
- sentinel error、`fmt.Errorf("...: %w", err)`、`errors.Is`。
- `time.Duration` 与 fail fast。
- 为什么不能在领域代码中随时调用 `os.Getenv`。

## 首次运行

```powershell
go test ./internal/platform/config -v
```

## 提示（按需展开）

1. 先构造默认 `Config`，再覆盖非空环境变量。
2. 必填值与格式值分别验证。
3. 测试给的是 map-backed 函数，不要改为直接调用 `os.Getenv`。

## 通关测试

```powershell
go test ./internal/platform/config -v
go test ./internal/platform/config -run TestLoadFrom -count=20
```

## 完成标准

- 所有现有测试及你补的边界测试通过。
- 能解释配置为什么只在进程入口解析一次。
- 更新 `my-progress.md`，记录一个错误包装示例。
