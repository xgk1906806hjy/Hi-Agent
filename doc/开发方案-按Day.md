# Hi-agent 开发方案（按 Day）

> 依据 [七天从零实现 GeekAgent](https://geektutu.com/books/geekagent) 全系列能力路线，按 Day **里程碑**增量落地本仓库。  
> **Day 只表示开发阶段，不建 `day1/`、`day2/` 目录。**  
> **实现语言：Go**；代码统一落在 `cmd/` + `internal/`。  
> 变更管理走 OpenSpec；模块文档在 `docs/`（改代码前必读，见 `AGENTS.md`）。

## 1. 总原则

| 原则 | 做法 |
|------|------|
| Day = 里程碑 | 按 Day 执行开发与验收，代码进统一工程，不按 Day 建目录 |
| 只增量不推翻 | 每个里程碑可运行；在现有模块上叠加能力 |
| history = messages | 多轮 / 工具 / system / 压缩都写入同一数组 |
| 能力分层 | 本地工具 → 横切能力 → 外部扩展 |
| 验收先于扩展 | 每天有明确验收；通过再进下一天 |
| Spec 先行 | 每个 Day（或阶段）先 `/opsx-propose`，再 `/opsx-apply` |

工程目录：

```
Hi-agent/
├── cmd/hi-agent/      # 程序入口
├── internal/          # 业务模块（config/color/chat/… 按能力分包）
├── doc/               # 本开发方案（按 Day 里程碑）
├── docs/              # 按模块：功能逻辑.md + 改动记录.md
├── openspec/          # OpenSpec 规格与变更
├── go.mod
├── .env.example
└── AGENTS.md
```

## 2. 阶段划分

| 阶段 | Days | 出口 |
|------|------|------|
| 一、基础 | 1–4 | 能聊、调工具、读写文件 |
| 二、会话与可见 | 5–7 | 长对话压缩、多会话持久化、TUI 用量 |
| 三、安全可控 | 8–9 | 权限/undo、TODO 与子 Agent |
| 四、记忆检索 | 10–14 | AGENTS.md、记忆、Skills、搜索、主动记忆、RAG |
| 五、扩展互联 | 15–16 | MCP、插件框架 |

推荐排期：W1 → Day1–4；W2 → Day5–8；W3 → Day9–12；W4 → Day13–16。  
每天节奏：**OpenSpec 提案（若未建）→ 实现 → 验收 → git tag `dayN`**。

---

## 3. Day 明细

### Day 1 — REPL 地基

- **目标**：流式多轮对话可跑（**Go**）
- **交付**：`internal/config`、`internal/color`、`internal/chat`、`cmd/hi-agent`
- **关键设计**：`history` 即 OpenAI messages；流式回调；Scanner 单线程防串台
- **验收**：`go run ./cmd/hi-agent` → 记住姓名 → `/reset` 清空
- **不做**：工具、持久化

### Day 2 — 工具调用循环

- **目标**：模型能「下单 → 执行 → 回传」
- **交付**：新增 `internal/tools`；扩展 `internal/chat`（工具循环最多 5 轮）
- **关键设计**：无关键词路由；流式拼装 `tool_calls`；缺 id 补 `call_N`
- **验收**：问「现在几点」→ `get_current_time` → 用真实时间回答
- **不做**：权限、危险工具

### Day 3 — Bash 工具

- **目标**：碰机器 + 三道护栏
- **交付**：`run_shell`（超时 10s / 截断 2k / `confirm` 可注入）
- **关键设计**：确认在工具 `run` 内，循环无感知
- **验收**：`ls` 前 `[y/N]`；`sleep 11` 超时；`n` 取消
- **不做**：目录隔离、allow/deny 表

### Day 4 — 文件工具 + 注册表

- **目标**：读免确认、写见 diff、统一注册
- **交付**：`ls/read/glob` + `write/patch`；`registerTool` 私有注册表
- **关键设计**：`patch` 唯一原文锚点；diff = 去公共前后缀
- **验收**：免确认读；`write/patch` 展示 diff 后落盘
- **里程碑**：读 → 改 → 验证闭环

### Day 5 — 历史压缩

- **交付**：超阈值摘要旧消息；保留最近 6 条；`/compact`
- **验收**：`GEEKAGENT_MAX_HISTORY=600` 聊爆后自动压缩且旧事实仍可答
- **不做**：二次摘要保护、轮内中途压缩

### Day 6 — 多会话持久化

- **交付**：`sessions.ts` + `storage.ts`；`/new` `/open` `/sessions` `/save` `/load`
- **关键设计**：一个 Chat + Map；`default` 退出改 8 位 ID
- **验收**：切会话不串话；重启 `/load` 恢复
- **落盘**：`.geekagent/sessions.json`

### Day 7 — 轻量 TUI + 用量

- **交付**：左消息 / 右面板；`stream_options.include_usage`
- **面板**：模型、会话、上下文占用、本轮/累计 tokens
- **验收**：流式时面板跳；压缩后占用回落；退出恢复终端

### Day 8 — 权限与回滚

- **交付**：`permissions.ts`（allow/ask/deny + safePath + redact）；`undo.ts`
- **配置**：`.geekagent/GeekAgent.json`
- **验收**：deny 拒绝；越界失败；密钥脱敏；`/undo` 恢复最近写入
- **不做**：shell 目录隔离、多步 undo

### Day 9 — 任务规划与子 Agent

- **交付**：`todo_write`（右栏进度）；`delegate_task`（干净上下文、无工具）
- **验收**：多步任务有 TODO；委派不污染主 history
- **约束**：同时最多一项 `in_progress`

### Day 10 — 项目指令 + 长期记忆

- **交付**：`AGENTS.md` 入每轮 system；`memory_write` / `memory_search` → `memory.json`
- **验收**：新会话仍遵守规则；跨会话可搜偏好；`/memory`
- **不做**：embedding、记忆编辑删除

### Day 11 — Skills

- **交付**：`skills/*/SKILL.md` + 可选 `tools.ts`；`/use` `/unuse` `/skills`
- **示例**：`code-review`、`explore`
- **验收**：加载后工具清单变化；卸载恢复默认

### Day 12 — search + fetch

- **交付**：仓库内容搜索（路径:行号）；网页转纯文本（先确认）
- **验收**：搜符号定位；fetch 返回无标签正文
- **不做**：ripgrep、完整 HTML 解析库

### Day 13 — 主动记忆

- **交付**：切块 + BM25；每轮提问前自动 Top5 进 system
- **验收**：不问「回忆」也能带上相关记忆

### Day 14 — 轻 RAG

- **交付**：`rag_add` / `rag_search` / `/rag`；索引 `.geekagent/rag/index.json`
- **验收**：采集后提问带来源段号；采集全文不进当前上下文

### Day 15 — MCP 客户端

- **交付**：`.geekagent/mcp.json`；stdio；`mcp_<server>_<tool>` 注册
- **验收**：`/mcp` 显示连接；模型调外部工具；权限走 Day8

### Day 16 — 插件框架

- **交付**：扫描 `plugins/*/plugin.ts`；`PluginContext`；`onStart`/`onExit`
- **示例**：`echo`、`web`（SSE）
- **验收**：`/plugins`；`/echo`；web 模式浏览器可聊

---

## 4. 横切约定

1. **命令**：内置 `switch` 优先，未命中再交插件命令表
2. **权限**：未知工具默认 `deny`；新增工具必须补进默认策略表
3. **输出**：进度行黄（工具/压缩），模型绿，系统灰
4. **状态目录**：一律 `.geekagent/`（gitignore）
5. **配置**：`OPENAI_BASE_URL` / `OPENAI_API_KEY` / `OPENAI_MODEL` 一份 `.env` 全仓共用

## 5. 与 OpenSpec 的映射

| 工作方式 | 说明 |
|----------|------|
| 主规格域 | 建议按能力拆：`repl`、`tools`、`sessions`、`permissions`、`memory`、`skills`、`rag`、`mcp`、`plugins` |
| 变更命名 | 推荐 `day01-repl`、`day02-tool-loop` … 或按阶段 `phase1-foundation` |
| 日常流程 | `/opsx-explore`（可选）→ `/opsx-propose` → 审阅 → `/opsx-apply` → `/opsx-archive` |
| 归档后 | delta 合并进 `openspec/specs/`，本 `doc/` 方案作路线图参考，以 specs 为准 |

## 6. 当前进度

**已完成：Day 1–7**（REPL、工具循环、Bash、文件工具、历史压缩、多会话、轻量 TUI + 用量）。  
工具循环护栏（轮次 / 重复熔断 / 预算 / 收缩）已作为横切能力落地。  

**下一步：Day 8**（权限 allow/ask/deny、路径越界、密钥脱敏、`/undo`），在现有 `internal/` 上增量实现，不新建 day 目录。
