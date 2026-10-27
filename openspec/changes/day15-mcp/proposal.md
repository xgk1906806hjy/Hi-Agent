# Change: Day15 MCP stdio client

## Intent

接入 MCP（Model Context Protocol）stdio server：项目通过 `.geekagent/mcp.json` 配置外部工具服务器，启动时连接并把 tools 注册为 `mcp_<server>_<tool>`；模型可像普通工具一样调用，权限仍由 Day 8 的 allow/ask/deny 决定。

## Scope

- `internal/mcp`
  - 读取 `.geekagent/mcp.json`（兼容 `mcpServers` / `servers`）。
  - 启动 stdio 子进程；JSON-RPC `initialize`、`tools/list`（分页）、`tools/call`。
  - 动态注册 `mcp_<server>_<tool>`，支持关闭/重载时注销。
  - 连接状态、工具数、错误信息；单 server 失败不影响其它 server。
- `internal/tools`
  - 新增 `Unregister`，供 MCP 关闭/重载移除动态工具。
- `cmd/hi-agent`
  - 启动时加载 MCP；`/project` 重载；`/mcp` 查看状态；`/mcp reload` 重连；退出关闭子进程。
- `internal/ui` / `internal/gui`
  - `Status.MCP` 状态行，GUI 侧栏显示连接状态。
- `internal/permission`
  - 不改默认 allow；MCP 动态工具名默认 deny，用户需在 `GeekAgent.json` 放行。
- 文档、测试、构建。

## Non-goals

- 不做 HTTP/SSE transport、OAuth、resources/prompts/sampling。
- 不做自动重连、后台轮询或 tools/list 热更新。
- 不自动放行 MCP 工具；权限继续走 Day 8。
- 不实现 Day 16 插件框架。

## 验收

- 配置一个 stdio server 后，`/mcp` 显示已连接与工具数。
- server 的 tool 以 `mcp_<server>_<tool>` 出现在工具注册表，模型可调用并得到文本结果。
- 未在权限配置放行的 MCP 工具调用返回拒绝；放行后可调用。
- `/project` 切目录后重新加载新项目 MCP，旧动态工具不残留。
- 配置缺失不报错；配置损坏/启动失败有明确状态，不阻断其他工具与对话。
