# Hi-agent

用 **Go** 实现最小 Agent/Harness。对照 GeekAgent **按 Day 推进能力**（里程碑），**不建 dayN 目录**；代码在统一工程下增量演进。

装到 PATH 后，在**任意项目目录**执行 `hi-agent`，即可读写、修改该目录下的代码（相对当前工作目录）。

---

## 怎么运行（最短路径）

```text
安装 hi-agent → 填好 .env → 跑 setup-env → 重开终端 → cd 到项目 → hi-agent
```

```mermaid
flowchart TD
  A[安装 hi-agent 到 PATH] --> B[复制 .env.example 为 .env 并填写 Key]
  B --> C[运行 setup-env 脚本或 hi-agent setup-env]
  C --> D[重开终端]
  D --> E[cd 到要改的项目目录]
  E --> F[执行 hi-agent]
  F --> G[加载：进程环境 / 项目.env / 全局 ~/.hi-agent/.env]
  G --> H[以 cwd 为工作区启动 TUI]
  H --> I{输入}
  I -->|对话| J[流式回复 + 工具读写/改文件]
  I -->|斜杠命令| K[/pwd /status /sessions …]
  J --> L[确认写文件或 shell 后继续]
  L --> I
  K --> I
  I -->|/exit 或空闲 Ctrl+C| M[保存 .geekagent/sessions.json 并退出]
```

### 1. 安装

推荐用安装脚本（会 `go install`，并把 `GOPATH/bin` **自动写入用户 PATH**）：

```bash
# 先克隆
git clone https://github.com/xgk1906806hjy/Hi-Agent.git
cd Hi-Agent

# Windows
.\scripts\install.ps1

# Linux / macOS
chmod +x ./scripts/install.sh && ./scripts/install.sh
```

也可手动：`go install ./cmd/hi-agent`（或 `go install github.com/xgk1906806hjy/Hi-Agent/cmd/hi-agent@latest`），此时需自己保证 `$(go env GOPATH)/bin` 在 PATH 里。

安装脚本跑完后**重开一次终端**，即可直接敲 `hi-agent`。

### 2. 配置环境变量（填 `.env` + 一键脚本）

不用手动去系统设置里配变量。在仓库根目录：

```bash
# 1) 生成并编辑 .env（至少填 OPENAI_API_KEY）
cp .env.example .env          # Windows: copy .env.example .env

# 2) 一键写入全局配置 + 用户环境变量
# Windows
.\scripts\setup-env.ps1
# 或 .\scripts\setup-env.cmd

# Linux / macOS
chmod +x ./scripts/setup-env.sh && ./scripts/setup-env.sh

# 已安装二进制时（任意目录）
hi-agent setup-env            # 默认读当前目录 .env
hi-agent setup-env /path/to/.env
```

脚本会做两件事：

| 动作 | 位置 |
|------|------|
| 复制整份 `.env` | `~/.hi-agent/.env`（Windows：`%USERPROFILE%\.hi-agent\.env`） |
| 持久化 `OPENAI_API_KEY` / `OPENAI_BASE_URL` / `OPENAI_MODEL` | Windows 用户环境变量；Linux/macOS → `~/.hi-agent/env.sh` 并挂到 shell rc |

**配置完成后请关闭并重新打开终端**，再继续下一步。

启动时配置加载优先级（高 → 低）：进程已有环境变量 → 当前目录向上的 `.env` → 全局 `~/.hi-agent/.env`。

> 也可以不跑 setup-env，只在某个项目下放 `.env`；适合单项目。要「任意目录都能跑」请用上面的一键脚本。

### 3. 启动

```bash
cd /path/to/your/project
hi-agent
```

纯文本模式：`GEEKAGENT_TUI=0 hi-agent`

启动后会看到工作目录、模型、会话 ID、已加载的 `.env` 路径。工具读写基准就是**当前 cwd**——先 `cd` 再启动；换项目就退出后换目录再开。

### 4. 使用中

1. 在 `You ›` 用自然语言提需求。
2. 工具相对当前目录：`ls` / `glob` / `read` / `write` / `patch` / `run_shell`（写文件与 shell 需确认）。
3. 回复中 **Ctrl+C** 只中断本轮；空闲时 **Ctrl+C** 或 `/exit` 保存并退出。
4. 斜杠命令见下表。

---

## 斜杠命令速查

| 命令 | 作用 |
|------|------|
| `/help` | 帮助 |
| `/pwd` | 显示工作目录与会话文件路径 |
| `/status` | 工作目录、会话、tokens、护栏状态 |
| `/sessions` | 列出本项目全部会话（`*` 为当前） |
| `/new [id]` | 新建空会话并切换 |
| `/open <id>` | 切换到已有会话 |
| `/save` | 立刻写入 `.geekagent/sessions.json` |
| `/load` | 从磁盘重载（丢弃未保存改动） |
| `/compact` | 压缩旧对话摘要 |
| `/reset` | 清空当前会话记忆 |
| `/exit` | 保存并退出 |

---

## 工作区与落盘约定

| 内容 | 位置 |
|------|------|
| 读/写/改文件、跑 shell | **启动时的当前工作目录（cwd）** |
| 多会话 JSON | `{cwd}/.geekagent/sessions.json` |
| 全局 API 配置 | `~/.hi-agent/.env` |
| 项目可选配置 | `{项目}/.env` 或上级目录 `.env` |

`.geekagent/` 与 `.env` 已在 `.gitignore` 中，勿把密钥提交进仓库。

---

## 当前能力（Day1–7）

- 流式多轮对话（OpenAI 兼容接口）
- 工具调用循环：`get_current_time`、`run_shell`、`ls`、`glob`、`read`、`write`、`patch`
- 危险操作确认（shell / 写文件展示 diff）
- 历史自动压缩与 `/compact`
- 工具循环护栏：轮次上限、重复熔断、回复预算、后期只读工具集、强制收尾
- 多会话持久化：`.geekagent/sessions.json`
- 轻量 TUI + token 用量面板
- 全局 CLI：任意 cwd 启动，分层加载 `.env`
- 一键环境配置：`setup-env` / `scripts/setup-env.*`

## 工程结构

```
cmd/hi-agent/       # 入口（REPL）
internal/config/    # 环境配置（全局 + 项目 .env）
internal/color/     # 终端着色
internal/chat/      # 对话、流式、工具循环、压缩、用量
internal/tools/     # 工具注册与执行（相对 cwd）
internal/session/   # 多会话与落盘（项目 .geekagent/）
internal/tui/       # 轻量全屏界面与面板
scripts/            # install + setup-env（Windows / Linux）
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
