# Proposal: Day2 工具调用循环

## Intent

让模型能通过 function calling 请求本地工具，程序执行后把结果回传，再继续生成最终回答。

## Scope

- 新增 `internal/tools`：`get_current_time`
- 扩展 `internal/chat.StreamReply`：最多 5 轮工具循环
- REPL 对工具进度行黄色着色

## Non-goals

- 权限、确认、shell、文件工具（后续 Day）
