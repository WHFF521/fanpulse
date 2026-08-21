# 第 04 关：HTTP JSON、错误契约与 OpenAPI

## 业务背景

API 不只是 handler 能返回数据；状态码、Content-Type、错误结构、认证和幂等头都是客户端依赖的契约。

## 需要完成的代码

文件：`internal/httpapi/problem.go`

### `WriteJSON(w, status, value) error`

- 在 `WriteHeader` 前设置 `Content-Type: application/json`。
- 使用 `json.NewEncoder(w).Encode(value)` 输出一个 JSON 值。
- 认识到 header 写出后无法安全改成 500；在学习日志中写下生产实现如何缓冲小响应。

### `ProblemFor(err, requestID) Problem`

- `ErrValidation` 映射为 400/`VALIDATION_FAILED`。
- 未知错误映射为 500/`INTERNAL_ERROR`，客户端 detail 只写通用描述。
- 使用 `errors.Is` 支持包装错误。
- `Type` 使用 `https://fanpulse.dev/problems/<slug>`，始终回传 request ID。

### OpenAPI 文件

创建 `api/openapi/fanpulse.yaml`，至少定义：

- `GET /health/live`
- `GET /api/v1/events`
- `POST /api/v1/events/{event_id}/participants`
- `POST /api/v1/rewards/{reward_id}/claims`
- `Problem` schema 和 `Idempotency-Key` header。

语义以 `docs/api-contract.md` 为准。不要一次抄完所有未来接口；先保证本关接口正确。

## 涉及知识

- `net/http`、`httptest.ResponseRecorder`、JSON 编码。
- RFC 9457 Problem Details。
- 契约优先、兼容性、API 版本。
- 内部错误与公开错误的安全边界。

## 通关测试

```powershell
go test ./internal/httpapi -v
go test ./coursechecks -run TestLevel04OpenAPIContract -v
```

推荐安装 OpenAPI linter 后再运行规范校验；静态关卡测试只能发现缺失内容，不能证明 YAML 合法。

## 完成标准

- 单元测试和契约检查通过。
- 为包装后的 `ErrValidation` 补测试。
- 能用 curl 展示一个成功 JSON 和一个 Problem JSON（HTTP server 接线可留到第 12 关）。
