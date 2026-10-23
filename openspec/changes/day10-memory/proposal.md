# Change: Day10 项目指令与长期记忆

## Intent

每轮发给模型的请求都带上工作区 `AGENTS.md`，但不把它写进会话 history。项目里可以追加长期记忆，并在以后的会话里按子串搜到。

## Scope

- 请求 messages 最前面附上 `AGENTS.md`；缺失则跳过；约 12000 字截断
- 新模块 `internal/memory`：`.geekagent/memory.json` 为 `{id, content, createdAt}[]`
- 工具 `memory_write`（追加）、`memory_search`（不区分大小写，最多 5 条），默认 allow
- REPL `/memory` 列出全部；`/help` 与斜杠补全同步
- 切换项目后读新目录的 `AGENTS.md` 和记忆文件

## Non-goals

- embedding、记忆的修改和删除
- Day 11 及之后的能力

## 验收

- 换会话后下一轮请求仍带当前 `AGENTS.md`，会话文件里没有这段 system
- `memory_write` 后换一个会话，`memory_search` 能搜到
- `/memory` 列出全部
