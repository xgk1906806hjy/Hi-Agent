# Proposal: Day6 多会话持久化

## Intent

支持多个互不串话的对话会话，内存 Map 管理，落盘 `.geekagent/sessions.json`，重启可 `/load` 恢复。

## Scope

- 新模块 `internal/session`（存储 + 会话管理）
- `chat` 导出 `History` / `SetHistory`
- REPL：`/new` `/open` `/sessions` `/save` `/load`；退出时若 `default` 有内容则改名为 8 位 ID 并自动保存
- 启动时若存在落盘文件则自动加载

## Non-goals

- TUI / token 面板（Day7）
- 云同步、加密、会话标题编辑
- 按 Day 建目录
