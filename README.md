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
  G --> H[以 cwd 为工作区启动界面：TUI 默认 / plain / GUI 桌面窗 / Web 浏览器]
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

**终端版**（`go install` / 安装脚本）只需 Go，不需要 Node 或 C 编译器。

**桌面 GUI 版**（`scripts/build-gui.*`）额外需要：

| 系统 | 需要 |
|------|------|
| 通用 | Node 18+（仅打包前端时） |
| Windows | WebView2（Windows 10/11 通常自带） |
| Linux | `webkit2gtk` 开发包，如 Debian/Ubuntu：`sudo apt install libwebkit2gtk-4.1-dev` |
| macOS | Xcode Command Line Tools |

更新代码后重新执行安装脚本（或 `go install ./cmd/hi-agent`）覆盖终端版；桌面版需再跑一次 `build-gui`。

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

**模型供应商**（`GEEKAGENT_PROVIDER`）：内置 `deepseek`（默认）、`openai`、`moonshot`、`qwen`、`zhipu`、`siliconflow`、`openrouter`、`groq`、`ollama`；中转站用 `custom` 并自填 `OPENAI_BASE_URL`。也可用厂商专用 Key（如 `DEEPSEEK_API_KEY`）。详见 `/help` 或 `.env.example`。

> 也可以不跑 setup-env，只在某个项目下放 `.env`；适合单项目。要「任意目录都能跑」请用上面的一键脚本。

### 3. 启动

```bash
cd /path/to/your/project
hi-agent
```

启动后会看到工作目录、模型、会话 ID、已加载的 `.env` 路径。工具读写基准就是**当前 cwd**——先 `cd` 再启动；换项目就退出后换目录再开。

#### 选择界面

用 `GEEKAGENT_UI`（或 `HI_AGENT_UI`）切换，默认 `tui`：

| 值 | 界面 | 说明 |
|----|------|------|
| `tui`（默认） | 终端全屏 | 左侧对话、右侧用量面板 |
| `plain` | 纯文本终端 | 兼容旧变量 `GEEKAGENT_TUI=0` |
| `gui` | 桌面窗口（Wails + React） | 须用 `scripts/build-gui` 打出的桌面版二进制；精密纸面 + 标题栏、项目/会话侧栏、Markdown 对话流 |
| `web` | 本机浏览器（HTTP + SSE） | 终端版设 `GEEKAGENT_UI=web`；默认 http://127.0.0.1:8787，`GEEKAGENT_WEB_ADDR` 可改监听地址 |

终端版二进制不含桌面界面：对它设 `GEEKAGENT_UI=gui` 会提示用 build-gui 构建。桌面版 exe 已通过 ldflags 默认进入 GUI；Web 模式请用终端版启动。

从源码直接运行终端界面：`go run ./cmd/hi-agent`。

#### 双击运行（桌面版 exe）

不想开终端，可打包成双击就弹窗口的 exe（没有黑色控制台窗口）：

```powershell
# Windows，仓库根目录（需 Node 18+）
.\scripts\build-gui.ps1                         # 生成 dist\Hi-agent.exe
.\scripts\build-gui.ps1 -Desktop                # 复制到桌面 + 快捷方式
.\scripts\build-gui.ps1 -Desktop -WorkDir D:\code\myproj   # 快捷方式默认打开指定项目
```

> **别和终端命令搞混**：在终端敲 `hi-agent` 走的是 PATH 里的**终端版**（默认 TUI）。  
> 桌面窗请双击 `dist\Hi-agent.exe`、桌面上的 `Hi-agent.exe`，或 `Hi-agent-gui.exe`。

```bash
# Linux / macOS：生成默认进 GUI 的 dist/hi-agent-gui（需 Node 18+）
./scripts/build-gui.sh
```

工作目录（Agent 读写文件的范围）怎么定：

| 启动方式 | 工作目录 |
|----------|----------|
| 双击 `Hi-agent.exe` | exe 所在文件夹 |
| 双击桌面快捷方式 | 快捷方式的「起始位置」（右键 → 属性可改） |
| 把项目文件夹拖到 exe 上 | 被拖的那个文件夹 |

模型配置仍读 `~/.hi-agent/.env`（先按上文跑过 `setup-env` 即可）。没配置也能打开窗口，按提示点 `/model` 填写并选择写入全局配置即可。exe 可以单独复制到任意位置使用。

### 4. 使用中

1. 用自然语言提需求（TUI/plain 在 `You ›` 后输入；GUI 在窗口底部输入框输入，点「发送」）。
2. 工具以当前目录为工作区：`ls` / `glob` / `read` / `search` / `write` / `patch` / `run_shell`，抓取网页的 `fetch`，以及轻 RAG 的 `rag_add` / `rag_search`（写文件、shell 与 fetch 需确认；search / rag_search 免确认）。
3. 斜杠命令见下表；输入 `/` 可展开命令菜单。
4. MCP：在项目 `.geekagent/mcp.json` 配置 stdio server（`mcpServers` 或 `servers`），启动时连接；工具以 `mcp_<server>_<tool>` 注册，用 `/mcp` 查看连接。MCP 工具默认 `deny`，需在 `.geekagent/GeekAgent.json` 的 `allow` 中放行具体工具或 `mcp_<server>_*`。
5. 插件：`plugins/<名称>/` 的 Go 包在 `init()` 中注册，由 `plugins/all` 编译期导入；用 `/plugins` 查看，`/echo <文本>` 是示例命令。
6. Web：`GEEKAGENT_UI=web hi-agent`，浏览器打开终端打印的地址即可聊天；默认只监听回环地址。

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "."]
    }
  }
}
```

详细协议与边界见 [docs/mcp/功能逻辑.md](docs/mcp/功能逻辑.md)。

各界面的操作差异：

| 操作 | TUI / plain | GUI | Web |
|------|-------------|-----|-----|
| 斜杠命令 | 输入 `/`（TUI 支持 Tab 补全） | 输入框输入，或点底部快捷按钮 | 输入框输入，或点侧栏命令 |
| 菜单选择（`/model` 等） | ↑/↓ 选择，Enter 确认 | 弹出列表，点选后按「选择」 | 浏览器 prompt 输入序号 |
| 危险操作确认 | 输入 `y` / `n` | 弹窗点「确认」/「取消」 | 浏览器 confirm |
| 查看历史消息 | PgUp/PgDn 或鼠标滚轮 | 聊天区直接滚动 | 聊天区直接滚动 |
| 中断当前回复 | Ctrl+C | 点输入框旁的「停止」 | 点「停止」 |
| 退出 | `/exit` 或空闲 Ctrl+C | `/exit` 或直接关闭窗口 | `/exit`（关闭标签不会结束进程） |

退出时都会自动保存会话到 `.geekagent/sessions.json`。

---

## 斜杠命令速查

| 命令 | 作用 |
|------|------|
| `/` | 展开命令菜单 |
| `/model` | 配置供应商 / 接口地址 / 模型 / Key，可写入全局 `.env` |
| `/help` | 帮助 |
| `/pwd` | 显示工作目录与会话文件路径 |
| `/status` | 工作目录、会话、tokens、护栏状态 |
| `/project <目录>` | 切换工作目录并加载该项目会话 |
| `/skills` | 列出 `skills/` 下的技能（`*` 为已加载） |
| `/use <名称>` | 加载技能：追加 system 片段并按 `tools` 限定工具 |
| `/unuse [名称]` | 卸载技能（省略名称则全部卸载） |
| `/sessions` | 列出本项目全部会话（`*` 为当前） |
| `/new [id]` | 新建空会话并切换 |
| `/open <id> [目录]` | 切换会话；带目录时先切项目 |
| `/rename` / `/delete` | 重命名 / 删除会话 |
| `/save` | 立刻写入 `.geekagent/sessions.json` |
| `/load` | 从磁盘重载（丢弃未保存改动） |
| `/compact` | 压缩旧对话摘要 |
| `/reset` | 清空当前会话记忆 |
| `/memory` | 列出项目长期记忆 |
| `/rag` | 查看 RAG 索引；`/rag clear` 清空索引 |
| `/mcp` | 查看 MCP 连接；`/mcp reload` 重读 `.geekagent/mcp.json` |
| `/plugins` | 查看编译期插件与生命周期；`/plugins reload` 重新执行 `onExit` / `onStart` |
| `/echo <文本>` | 示例插件命令（内置命令未命中时再查插件命令表） |
| `/undo` | 撤销最近一次 write/patch（最多 20 步） |
| `/permissions` | 显示项目与用户权限规则 |
| `/exit` | 保存并退出 |

---

## 工作区与落盘约定

| 内容 | 位置 |
|------|------|
| 读/写/改文件、跑 shell | **启动时的当前工作目录（cwd）**（路径约束在工作区内） |
| 多会话 JSON | `{cwd}/.geekagent/sessions.json` |
| 任务清单 | `{cwd}/.geekagent/todos.json`（按会话） |
| 长期记忆 | `{cwd}/.geekagent/memory.json` |
| RAG 索引 | `{cwd}/.geekagent/rag/index.json` |
| MCP 配置 | `{cwd}/.geekagent/mcp.json`（stdio server；不入库） |
| 权限规则 | `{cwd}/.geekagent/GeekAgent.json`；用户级 `~/.hi-agent/GeekAgent.json` |
| 撤销栈 | `{cwd}/.geekagent/undo.json` |
| 全局 API 配置 | `~/.hi-agent/.env` |
| 项目注册表 | `~/.hi-agent/projects.json`（GUI） |
| 项目可选配置 | `{项目}/.env` 或上级目录 `.env` |
| 项目指令 | 工作区根 `AGENTS.md`（每轮注入，不进会话 history） |
| 技能 | `{cwd}/skills/<名称>/SKILL.md`（按需 `/use` 加载，不落盘） |

`.geekagent/` 与 `.env` 已在 `.gitignore` 中，勿把密钥提交进仓库。

---

## 当前能力（Day 1–16）

- 流式多轮对话（OpenAI 兼容接口）与工具循环护栏
- 工具：`get_current_time`、`run_shell`、`ls` / `glob` / `read` / `write` / `patch`、`todo_write`、`delegate_task`、`memory_write` / `memory_search`、`rag_add` / `rag_search`、`search` / `fetch`
- 仓库内容搜索（`路径:行号`，字面/正则，纯 Go）与网页抓取转纯文本（默认先确认）
- 轻 RAG：`rag_add` 采集工作区文件/目录到 `.geekagent/rag/index.json`；`rag_search` 用 BM25 返回带 `[来源#段号]` 的 TopK 片段，采集全文不回显到当前上下文
- MCP 客户端（stdio）：读取 `.geekagent/mcp.json`，握手并注册 `mcp_<server>_<tool>`；`/mcp` 查看连接，`/mcp reload` 重连；MCP 工具默认 deny，需权限配置放行
- 权限（allow/ask/deny）、路径越界、输出脱敏；多步 `/undo`；工作区路径约束（非 OS 沙箱）
- 任务清单按会话落盘；子任务委派（复制主 history、可用工具、仅一层）
- 项目指令 `AGENTS.md` + 长期记忆：显式 `memory_search` 子串搜索；每轮自动用 BM25 召回 Top5 记忆块注入临时 system，不写入 history
- 技能：`skills/*/SKILL.md` + `/use` `/unuse` `/skills`，system 片段与工具白名单
- 多会话持久化、轻量 TUI + token 面板
- 桌面 GUI（Wails + React）：精密纸面视觉、无边框标题栏、主题三态、跨项目会话管理、技能卡片、MCP 连接状态
- 编译期插件：`plugins/*` + `internal/plugins`，`onStart` / `onExit`、插件命令回退、插件工具；`/plugins`、`/echo`
- Web 模式：`GEEKAGENT_UI=web` 启动 HTTP + SSE 浏览器界面，复用 GUI Adapter 与事件协议
- 全局 CLI + `setup-env` 一键环境配置

**Day 16 已完成**（插件框架 + Web 模式）。路线图见 [doc/开发方案-按Day.md](doc/开发方案-按Day.md)。

## 工程结构

```
cmd/hi-agent/       # 入口（REPL）
internal/config/    # 环境配置（全局 + 项目 .env）
internal/color/     # 终端着色
internal/chat/      # 对话、流式、工具循环、压缩、用量、AGENTS、委派
internal/tools/     # 工具注册与执行（相对 cwd）
internal/session/   # 多会话与落盘（项目 .geekagent/）
internal/permission/# 权限、SafePath、脱敏
internal/plan/      # 任务清单（todos.json）
internal/memory/    # 长期记忆（memory.json、BM25 主动召回）
internal/rag/       # 轻 RAG（rag/index.json、切块 + BM25）
internal/mcp/       # MCP stdio 客户端（mcp.json、动态工具注册）
internal/plugins/   # 编译期插件注册、生命周期与命令分派
plugins/            # 插件示例（echo / web）与 all 汇总
internal/skills/    # 技能扫描（skills/*/SKILL.md）
internal/tui/       # 轻量全屏界面与面板
internal/ui/        # UI 抽象工厂（plain / tui / gui / web）
internal/gui/       # Wails 桌面对话壳（desktop 标签 + React 前端）
internal/webui/     # 浏览器界面（HTTP + SSE，复用 gui.Adapter）
internal/workspace/ # 项目注册表与会话列表
scripts/            # install + setup-env + build-gui（Windows / Linux）
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
