# Change: Day12 search + fetch

## Intent

给模型两个检索类工具：本地仓库按行搜索（`路径:行号`），以及抓取网页并转成纯文本。搜索免确认；联网抓取默认 ask，执行前必须经用户确认。

## Scope

- 新工具 `search`（默认 allow，计入只读集合）
  - 参数：`pattern`、可选 `path`、可选 `regex`
  - 纯 Go 遍历工作区，输出 `相对路径:行号: 行内容`
  - 跳过 `.git` / `.geekagent` / `node_modules` / `dist` 等目录、二进制与超大文件
  - 最多 200 条，超出提示收窄范围
- 新工具 `fetch`（默认 ask，必须内部确认）
  - 参数：`url`，仅 http/https
  - 超时 15s、响应上限 2MB、正文最多 8000 字
  - 去除 script/style/注释与 HTML 标签，解码实体，压缩空白；输出标题与正文
- `permission` 默认策略与规则主体：`search` 取 pattern，`fetch` 取 url
- `tools` 只读集合加入 `search`（工具集收缩后仍可用）
- 单测：搜索过滤/正则/范围/上限；fetch 转文本、确认取消、非 2xx、协议校验
- 文档与路线图进度

## Non-goals

- 不调用 ripgrep / 外部搜索命令
- 不引入完整 HTML 解析库，不执行 JavaScript，不处理登录/反爬
- 不做非 UTF-8 网页的字符集转码
- 不做搜索缓存、增量索引、RAG（Day 14）

## 验收

- `search` 能定位符号并给出 `路径:行号`；不返回 `.git` / `node_modules` 中的结果
- `regex: true` 可用；非法正则返回可读错误
- 默认配置下 `fetch` 先弹出确认；取消不发起请求
- 抓取 HTML 后返回无标签正文与标题；`<script>` / `<style>` 内容不出现
- 非 http/https、HTTP 非 2xx、超时均返回文本说明，工具循环不中断
