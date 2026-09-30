# Change: 工具循环护栏 P2

## Why

长任务仍可能堆大量工具输出或反复写文件；需字符预算与后期只读收缩，降低空转与误写风险。

## What Changes

- `GEEKAGENT_MAX_REPLY_CHARS`（默认 120000，0=关）→ 强制收尾
- `GEEKAGENT_SHRINK_TOOLS_AFTER`（默认约 3/4 轮次，0=关）→ 只暴露只读工具
- `tools.ReadonlyNames` / `ToOpenAIAllow`

## Impact

- `internal/chat/*`、`internal/tools/tools.go`、`cmd/hi-agent/main.go`
- 方案：`doc/工具循环优化方案.md`
