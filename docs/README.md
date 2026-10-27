# 模块文档（docs）

按**业务模块**维护（不是按 Day）。每个模块固定两个文件：

| 文件 | 用途 |
|------|------|
| `功能逻辑.md` | 职责、接口、**逐步逻辑流程**、环境变量、边界、依赖 |
| `改动记录.md` | 按时间追加的变更日志 |

智能体改代码前须先读对应模块上述两文件（见根目录 `AGENTS.md`）。

Day 只表示开发里程碑，见 `doc/开发方案-按Day.md`。代码统一在 `cmd/` + `internal/`。

## 模块索引

| 模块 | 说明 | 代码 | 文档要点 |
|------|------|------|----------|
| [config](./config/) | 模型与环境配置 | `internal/config/` | `.env` 优先级、`Load` 退出路径 |
| [color](./color/) | 终端着色与输出 | `internal/color/` | TTY 检测、四色约定 |
| [chat](./chat/) | 对话历史、流式与工具循环 | `internal/chat/` | StreamReply / 压缩 / 护栏 / 用量 |
| [tools](./tools/) | 工具注册与执行 | `internal/tools/` | 注册表、确认、各工具与 diff 落盘、search/fetch |
| [session](./session/) | 多会话内存与落盘 | `internal/session/` | Map、JSON、与 Chat 同步、退出改名 |
| [tui](./tui/) | 轻量全屏界面与面板 | `internal/tui/` | 左右分栏、用量面板、终端恢复 |
| [ui](./ui/) | UI 抽象与工厂（可插拔） | `internal/ui/` | `UI` 接口、Register/Create、Kind |
| [gui](./gui/) | Wails 桌面对话壳 | `internal/gui/` | 精密纸面、无边框标题栏、协议 / Adapter / React（desktop） |
| [workspace](./workspace/) | 项目注册表与会话列表 | `internal/workspace/` | `projects.json`、`ListSessions` |
| [permission](./permission/) | 工具权限、路径越界、脱敏（Day 8） | `internal/permission/` | `GeekAgent.json` 规则、`SafePath`、`Redact` |
| [plan](./plan/) | 任务清单（Day 9） | `internal/plan/` | 按会话写入 `todos.json`，同时最多一项 in_progress |
| [memory](./memory/) | 项目长期记忆（Day 10 / 13） | `internal/memory/` | `memory.json` 追加、子串搜索、切块 + BM25 主动召回 |
| [rag](./rag/) | 项目轻 RAG（Day 14） | `internal/rag/` | `.geekagent/rag/index.json`、切块 + BM25、来源段号 |
| [mcp](./mcp/) | MCP stdio 客户端（Day 15） | `internal/mcp/` | `.geekagent/mcp.json`、握手 / tools、动态注册、权限默认 deny |
| [skills](./skills/) | 技能加载（Day 11） | `internal/skills/` | `skills/*/SKILL.md`、system 片段与工具白名单 |
| [repl](./repl/) | REPL 与斜杠命令 | `cmd/hi-agent/` | 启动、命令表、经 ui 工厂创建界面 |

## 模块依赖（运行时）

```
cmd/hi-agent (repl)
    ├── config
    ├── color
    ├── workspace
    ├── ui ──► plain | tui(──► color) | gui(──► Wails/React，desktop 标签) | web(预留)
    ├── session ──► chat
    ├── chat ──► tools
    ├── permission（Load / SafePath / Redact → 注入 tools 钩子）
    ├── plan（任务清单；todo_write 写入，界面只读）
    ├── memory（长期记忆；memory_write / memory_search、/memory；RecallNote 经 SetRecall 注入 chat）
    ├── rag（轻 RAG；rag_add / rag_search、/rag；索引跟随 cwd）
    ├── mcp（stdio 客户端；mcp_<server>_<tool> 动态注册；/mcp；默认 deny）
    ├── skills（技能；skills/*/SKILL.md → system 片段 + 工具白名单）
    └── tools（SetConfirm / SetPolicy / SetPathGuard / SetRedactor / UndoLast）
```

## 相关文档

| 路径 | 内容 |
|------|------|
| `doc/开发方案-按Day.md` | Day 里程碑 |
| `doc/工具循环优化方案.md` | 护栏 P0–P2 |
| `openspec/` | 规格变更 |
| `.env.example` | OpenAI 相关变量示例 |
