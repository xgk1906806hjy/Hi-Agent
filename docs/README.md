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
| [chat](./chat/) | 对话历史、流式与工具循环 | `internal/chat/` | StreamReply / 压缩 / 护栏 / 收尾 / 中断 |
| [tools](./tools/) | 工具注册与执行 | `internal/tools/` | 注册表、确认、各工具与 diff 落盘 |
| [session](./session/) | 多会话内存与落盘 | `internal/session/` | Map、JSON、与 Chat 同步、退出改名 |
| [repl](./repl/) | REPL 与斜杠命令 | `cmd/hi-agent/` | 启动、命令表、着色分流、Ctrl+C |

## 模块依赖（运行时）

```
cmd/hi-agent (repl)
 ├── config
 ├── color
 ├── session ──► chat
 ├── chat ──► tools
 └── tools（SetConfirm / Names）
```

## 相关文档

| 路径 | 内容 |
|------|------|
| `doc/开发方案-按Day.md` | Day 里程碑 |
| `doc/工具循环优化方案.md` | 护栏 P0–P2 |
| `openspec/` | 规格变更 |
| `.env.example` | OpenAI 相关变量示例 |
