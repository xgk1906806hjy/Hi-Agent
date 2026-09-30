# Proposal: Day3 Bash 工具

## Intent

给模型接入 `run_shell`，在超时、输出截断、执行前确认三道护栏下执行本地命令。

## Scope

- `internal/tools`：`run_shell` + `SetConfirm`
- REPL 共享 stdin 注入确认
- chat 循环不改

## Non-goals

- 目录隔离、allow/deny 策略表、流式命令输出
