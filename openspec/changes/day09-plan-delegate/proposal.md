# Change: Day9 任务规划与子 Agent

## Intent

多步任务有一份可见的 TODO；需要独立思考的子任务交给一个干净的模型实例，不污染主对话 history，也不带工具。

## Scope

- 新模块 `internal/plan`：内存清单，同时最多一项 `in_progress`
- 工具 `todo_write`（按 ID 合并）、`delegate_task`（`chat.Delegate`：同一端点与模型，独立 messages，无 tools）
- 默认权限 allow（二者不改工作区文件）
- 展示：TUI 右栏「任务」、GUI 侧栏；`/project` 后清空清单
- `tools.ExecContext` 把当前回复的 context 传给委派，停止按钮可取消子请求

## Non-goals

- 清单落盘、按会话隔离
- 子 Agent 使用工具或看到主 history
- `AGENTS.md`、长期记忆（Day 10）

## 验收

- 两次把不同任务标为 in_progress，清单里只剩一项进行中
- 委派请求体不含主 history、不含 tools；返回文本不写入主 history
- 侧栏 / 右栏能看到清单
