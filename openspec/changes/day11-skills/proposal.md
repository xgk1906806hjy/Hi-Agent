# Change: Day11 Skills

## Intent

把 `skills/*/SKILL.md` 里的技能按需加载：给模型追加一段工作方式说明，并按技能声明的 `tools` 收窄可见工具集。技能只在当前进程生效，不改权限策略、不注册新工具。

## Scope

- 新模块 `internal/skills`：扫描 `skills/*/SKILL.md`，解析 `name` / `description` / `tools` 与正文
- REPL `/skills` `/use <名称>` `/unuse [名称]`；启动与 `/project` 后重新扫描
- `chat.SetSkills(note, allow)`：system 片段附在 `AGENTS.md` 之后；工具导出按白名单过滤
- 白名单与原有「只读收缩」取交集；`nil` 表示不限制、空切片表示一个都不给
- `/status` 与桌面侧栏展示已加载技能（`ui.Status.Skills` → `status.skills`）
- 示例技能 `skills/code-review/`、`skills/explore/`

## Non-goals

- TS 的 `tools.ts` 动态加载（Go 无运行时插件）
- 技能状态持久化、按会话区分
- 搜索 / fetch / MCP / RAG（Day 12 及之后）

## 验收

- `/use code-review` 后 requests 最前出现技能 system，会话文件中没有这段
- 加载声明了 `tools` 的技能后，模型可见工具只剩白名单；`/unuse` 后恢复默认
- 加载后桌面侧栏出现「技能」卡片，卸载后消失
- `/skills` 能列出仓库示例技能并标注加载状态
