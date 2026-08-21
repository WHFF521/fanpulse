# 第 02 关：用户领域模型与不变量

## 业务背景

邮箱唯一约束只有在应用与数据库使用相同规范化规则时才可靠。领域对象不应该存在“ID 为空、名称为空”这种半有效状态。

## 需要完成的代码

文件：`internal/identity/user.go`

### `NormalizeEmail(email string) string`

去除首尾空白并转换为小写。本项目明确不做提供商特定规则，例如删除 Gmail 的点或 `+tag`。

### `NewUser(id, email, displayName string, now time.Time) (User, error)`

要求：

- ID、显示名非空；显示名 trim 后长度 1～80。
- 邮箱先规范化，再用 `net/mail.ParseAddress` 或清晰的最小规则验证。
- 解析结果必须确实是单个邮箱，不能接受带显示名的整段输入作为存储值。
- `now` 统一转 UTC，或明确要求调用方传 UTC并测试。
- 失败返回包装后的 `ErrInvalidUser`。

建议新增测试：名称全是空格、超长名称、带空格邮箱、时间处理。

## 涉及知识

- 构造函数保护领域不变量。
- 值清洗与输入验证的区别。
- table-driven tests、子测试。
- 不要自创完整 RFC 邮箱正则的原因。

## 测试

```powershell
go test ./internal/identity -v
```

## 常见错误

- 只 `strings.ToLower`，忘记 trim。
- 把原始 email 放入返回对象。
- 返回普通新错误导致 `errors.Is(err, ErrInvalidUser)` 失败。
- 为通过当前三个用例而硬编码域名。

## 完成标准

- 全部测试通过，至少补 3 个边界测试。
- 能解释为什么 API DTO 与 `User` 领域对象不应是同一个结构。
