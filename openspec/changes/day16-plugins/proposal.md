# Change: Day16 编译期插件框架与 Web 模式

## Intent

把原 TS 方案中的 `plugins/*/plugin.ts` 动态扫描替换为 Go 编译期注册：插件在 `init()` 声明 `onStart` / `onExit`、斜杠命令与可选工具，REPL 启动时执行生命周期，内置命令未命中时再分派给插件命令表。同时落地 `ui.KindWeb`，复用 `internal/gui` 的 `Adapter` / `Snapshot` / 事件协议，用 HTTP + SSE 提供本机浏览器界面。

## Scope

- `internal/plugins`
  - `Register`、`Plugin` / `Command` / `Context`、`Host`、`Manager`。
  - 生命周期：启动 `OnStart`、退出 `OnExit`、`/plugins reload`。
  - 插件命令按名分派；未命中返回 `handled=false`。
  - 插件工具注册/注销；启动失败只记录状态，不阻断 REPL。
- `plugins/`
  - 示例 `echo`（`/echo`）与 `web`（`/web` 提示）；`plugins/all` 空白导入完成编译期聚合。
- `cmd/hi-agent`
  - 启动插件管理器；内置命令优先；`/plugins` / `/plugins reload`；退出关闭插件。
  - 斜杠菜单与 `/help` 同步插件命令。
  - `GEEKAGENT_UI=web` 使用 `config.LoadE`；`DesktopRunner` 分支覆盖 GUI / Web。
- `internal/webui`
  - 注册 `ui.KindWeb`；复用 `gui.Adapter`。
  - HTTP：`/`、`/api/events`（SSE）、`/api/snapshot`、`/api/submit`、`/api/stop`、`/api/answer`。
  - 内嵌单文件页面；默认监听 `127.0.0.1:8787`，支持 `GEEKAGENT_WEB_ADDR`。
- 文档、测试、README、路线图、OpenSpec。

## Non-goals

- 不做运行时动态加载、插件热插拔或从文件扫描 `.go`。
- 不做 Web 鉴权、HTTPS、多用户隔离；默认只绑回环地址。
- 不改变 Wails 桌面启动与 React 前端；Web 页面独立内嵌，复用协议而非构建产物。
- 不自动放行插件工具；权限继续走 Day 8，未知工具默认 deny。

## 验收

- `/plugins` 显示 `echo`、`web` 插件及启动状态；`/plugins reload` 重新执行生命周期。
- `/echo hello` 输出 `hello`；未知命令仍打印未知提示。
- `GEEKAGENT_UI=web hi-agent` 启动后在浏览器打开默认地址，可发送消息并看到流式回复；停止按钮可中断。
- Web 的 `/api/snapshot` 返回 `{status,nav,prompt,slash}`；SSE 事件名与 GUI 协议一致。
- `go test -p 1 ./...`、`go vet ./...` 通过。
