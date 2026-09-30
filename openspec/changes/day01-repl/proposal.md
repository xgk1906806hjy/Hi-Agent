# Proposal: Day1 REPL 地基

## Intent

建立 Hi-agent 最小可运行入口：终端 REPL + OpenAI 兼容流式多轮对话，为后续工具循环、会话与权限打地基。

对应路线：`doc/开发方案-按Day.md` → Day 1。

## Scope

- 初始化工程：`go.mod`、Go 依赖（go-openai / godotenv / x/term）
- 实现统一工程：`internal/config|color|chat`、`cmd/hi-agent`（不建 day 目录）
- 斜杠命令：`/help`、`/reset`、`/exit`
- `.env.example` 与根目录配置约定

## Approach

- Day 是里程碑，代码按模块进 `internal/` / `cmd/`
- `history` 直接使用 OpenAI `messages`，不做中间模型
- `StreamReply` 流式回调 `onDelta` 边收边打
- `bufio.Scanner` 搭 REPL（单线程处理）
- 终端着色集中在 `internal/color`

## Non-goals

- 工具调用、文件读写、会话持久化
- 自定义 TUI / 用量面板（Day 7）
- 权限与目录隔离（Day 8）
