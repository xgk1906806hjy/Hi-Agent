# Change: Day7 轻量 TUI + 用量

## Intent

终端左右分栏：左侧消息流、右侧状态面板（模型、会话、上下文占用、本轮/累计 tokens）；流式请求附带 `stream_options.include_usage`。

## Scope

- 新模块 `internal/tui`：备用屏幕、整屏重绘、右侧面板、底部输入
- `chat`：用量累计、上下文占用、`StreamOptions.IncludeUsage`、`SetUsageHook` / `Usage` / `Model`
- REPL：默认启用 TUI（非终端或 `GEEKAGENT_TUI=0` 回退纯文本）；空闲 Ctrl+C 正常退出并恢复终端

## Non-goals

- 权限 / undo（Day8）
- 真正的 raw-mode 多行编辑器、鼠标支持
- 精确 tokenizer（无用量时按 history JSON 粗估）
