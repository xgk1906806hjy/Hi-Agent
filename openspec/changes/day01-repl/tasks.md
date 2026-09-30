# Tasks: Day1 REPL 地基（Go，统一工程）

## 1. 工程脚手架

- [x] 1.1 `go.mod`（module `hi-agent`）
- [x] 1.2 依赖：`go-openai`、`godotenv`、`x/term`
- [x] 1.3 `.env.example`、`.gitignore`

## 2. 核心模块（非 day 目录）

- [x] 2.1 `internal/config`：`Load`
- [x] 2.2 `internal/color`：`Paint` / `Out` / `Err`
- [x] 2.3 `internal/chat`：`StreamReply` + `Reset`
- [x] 2.4 `cmd/hi-agent`：REPL、`/help` `/reset` `/exit`

## 3. 验收

- [x] 3.1 `go build ./cmd/hi-agent` 通过
- [ ] 3.2 `go run ./cmd/hi-agent` 可进入 `You ›`（需 `.env`）
- [ ] 3.3 多轮记忆与 `/reset`
- [ ] 3.4 `/exit` 打印 `bye`

## 4. 收尾

- [x] 4.1 文档与 AGENTS 标明：Day=里程碑，代码在 cmd/internal
- [ ] 4.2 归档 OpenSpec 后打 tag（可选）
