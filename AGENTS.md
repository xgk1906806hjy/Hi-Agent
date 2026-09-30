# Hi-agent 项目指令

本项目使用 **Go** 实现。对照 GeekAgent **按 Day 推进能力**（里程碑），**不按 Day 建目录**；代码落在统一工程结构下增量演进。

## 强制规则：改代码前先读模块文档

智能体在**修改任何模块相关代码之前**，必须先完整阅读该模块在 `docs/<模块名>/` 下的两个文件：

1. `docs/<模块名>/功能逻辑.md` — 功能与逻辑说明  
2. `docs/<模块名>/改动记录.md` — 历史改动与注意点  

**未读取上述文件，不得修改对应代码。**

修改完成后，须在同模块的 `改动记录.md` 末尾追加一条记录（日期、改动摘要、影响范围）。

### 模块与代码路径对照

| 模块 | 文档目录 | 代码路径 |
|------|----------|----------|
| config | `docs/config/` | `internal/config/` |
| color | `docs/color/` | `internal/color/` |
| chat | `docs/chat/` | `internal/chat/` |
| tools | `docs/tools/` | `internal/tools/` |
| session | `docs/session/` | `internal/session/` |
| repl | `docs/repl/` | `cmd/hi-agent/` |

新增模块时：在 `docs/<模块名>/` 下创建上述两个文件，代码放 `internal/<模块>/`（入口放 `cmd/hi-agent/`），并更新本表与 `docs/README.md`。

## 工程结构

```
Hi-agent/
├── cmd/hi-agent/     # 程序入口（REPL）
├── internal/         # 业务模块（按能力分包，不按 day 分包）
├── docs/             # 模块功能逻辑 + 改动记录
├── doc/              # 开发方案（按 Day 里程碑）
└── openspec/         # 规格与变更
```

## 其他约定

- 开发节奏见 `doc/开发方案-按Day.md`（Day = 交付阶段，不是目录名）。
- 规格变更走 OpenSpec（`openspec/`）。
- 只增量不推翻；不提前实现后续 Day 的能力。
- `history` 即 OpenAI Chat Completions 的 `messages`，零转换。
- 密钥与本地状态：`.env`、`.geekagent/` 不入库。
