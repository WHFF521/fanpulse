# 第 00 关：建立闯关环境

## 目标

能独立运行单个测试、解释红灯/绿灯含义，并创建自己的学习记录。此关不写业务代码。

## 你要做什么

1. 安装 Go，并执行 `go version`；版本必须满足 `go.mod`。
2. 阅读根 README、需求规格和架构设计的 1～7 节。
3. 复制 `challenges/my-progress.example.md` 为 `challenges/my-progress.md`，记录开始日期、Go 版本、操作系统和每关状态。
4. 运行仓库基线测试。

```powershell
go test ./coursechecks -run TestLevel00RepositoryBaseline -v
go test ./internal/platform/config -v
```

第一个命令应通过；第二个命令应失败在 `TODO(level-01)`。这就是正确初始状态。

## 涉及知识

- Go module、package、`*_test.go`、`go test -run`。
- 编译失败、测试失败和程序运行失败的区别。
- Arrange/Act/Assert 测试结构。

## 通关测试

```powershell
go test ./coursechecks -run TestLevel00RepositoryBaseline -v
```

## 通关前自问

- 为什么整个仓库的 `go test ./...` 现在必然失败？
- 如何只运行一个包、一个测试函数？
- 测试通过为什么不必然说明产品需求已经满足？

## 完成标准

- 基线测试通过。
- `my-progress.md` 已创建，且你能解释下一关的失败来自哪里。
