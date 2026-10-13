# Change: Day8 权限与回滚

## Intent

工具调用按 allow / ask / deny 判定；文件路径不得越出工作区；给模型看的输出与确认提示脱敏；`/undo` 恢复最近一次写入。

## Scope

- 新模块 `internal/permission`：读 `.geekagent/GeekAgent.json`，规则 `工具名` 或 `工具名:通配`，优先级 deny > ask > allow > 内置默认（只读 allow，`run_shell` / `write` / `patch` ask，未知 deny）
- `SafePath`（含符号链接）、`Redact`（密钥原值、`sk-`、`Bearer`、`*_API_KEY/TOKEN/SECRET=`）
- `tools` 钩子 `SetPolicy` / `SetPathGuard` / `SetRedactor`；`commitWrite` 记录 `.geekagent/undo.json`；`UndoLast`
- REPL：`/undo`、`/permissions`；`/project` 后重新 Load；配置损坏提示并回退默认，进程不退出

## Non-goals

- shell 目录隔离、多步 undo、用户级与项目级策略合并
- 权限配置热更新（见后续补强：按文件修改时间重新 Load）

## 验收

- deny 规则下对应工具返回「已拒绝」，不执行
- 工作区外路径（含符号链接逃逸）失败
- 工具输出与确认提示中的密钥被替换为 `***`
- `/undo` 恢复最近一次 write/patch；新建文件则删除
