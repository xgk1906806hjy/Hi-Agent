# Hi-agent

用 **Go** 实现最小 Agent/Harness。对照 GeekAgent **按 Day 推进能力**（里程碑），**不建 dayN 目录**；代码在统一工程下增量演进。

## 快速开始

```bash
cp .env.example .env   # 填写 OPENAI_API_KEY
go mod tidy
go run ./cmd/hi-agent
```

终端中默认进入 **TUI**（左侧消息 / 右侧用量面板）。纯文本模式：`GEEKAGENT_TUI=0 go run ./cmd/hi-agent`。

常用命令：`/help`、`/status`、`/sessions`、`/new [id]`、`/open <id>`、`/save`、`/load`、`/compact`、`/reset`、`/exit`。回复中 Ctrl+C 中断当前轮；空闲时 Ctrl+C 保存并退出。

## 当前能力（Day1–7）

- 流式多轮对话（OpenAI 兼容接口）
- 工具调用循环：`get_current_time`、`run_shell`、`ls`、`glob`、`read`、`write`、`patch`
- 危险操作确认（shell / 写文件展示 diff）
- 历史自动压缩与 `/compact`
- 工具循环护栏：轮次上限、重复熔断、回复预算、后期只读工具集、强制收尾
- 多会话持久化：`.geekagent/sessions.json`
- 轻量 TUI + token 用量面板（上下文占用、本轮/累计）

## 工程结构

```
cmd/hi-agent/       # 入口（REPL）
internal/config/    # 环境配置
internal/color/     # 终端着色
internal/chat/      # 对话、流式、工具循环、压缩、用量
internal/tools/     # 工具注册与执行
internal/session/   # 多会话与落盘
internal/tui/       # 轻量全屏界面与面板
docs/               # 模块功能逻辑 + 改动记录
doc/                # 按 Day 的开发方案（里程碑）
openspec/           # 规格驱动变更
```

## 文档

| 路径 | 说明 |
|------|------|
| [AGENTS.md](AGENTS.md) | 改模块前必读 docs |
| [docs/](docs/) | 按模块文档 |
| [doc/开发方案-按Day.md](doc/开发方案-按Day.md) | Day1–16 能力里程碑 |
| [doc/工具循环优化方案.md](doc/工具循环优化方案.md) | 工具循环护栏 |
| [doc/OpenSpec使用说明.md](doc/OpenSpec使用说明.md) | OpenSpec |

## License

[MIT](LICENSE)
