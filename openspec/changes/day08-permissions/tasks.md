# Tasks: Day8

- [x] `internal/permission`：Load / Decide / SafePath / Redact 与单测
- [x] `tools`：SetPolicy / SetPathGuard / SetRedactor；Exec 三态；undo.json 与 UndoLast
- [x] REPL：注入钩子、`/undo`、`/permissions`、`/project` 后重新 Load
- [x] 文档：`docs/permission`、AGENTS 模块表、tools / repl 改动记录
- [x] 验收：`go test ./internal/permission ./internal/tools`

## 后续补强（同日）

- [x] 工具调用前按 `GeekAgent.json` 修改时间自动重载，损坏只提示一次
- [x] GUI 顶栏「权限」展示当前规则
