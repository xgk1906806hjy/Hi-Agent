# Change: Day14 lightweight RAG

## Intent

为项目文档提供轻量、可持久化的检索增强：用户先把工作区文件/目录 `rag_add` 到 `.geekagent/rag/index.json`，之后模型用 `rag_search` 取回相关片段并带来源段号（如 `docs/guide.md#3`）。`rag_add` 只返回索引统计，不把采集全文放进当前上下文。

## Scope

- `internal/rag`
  - 文件/目录采集，跳过噪声目录、二进制、超大/空文件和符号链接。
  - 按段落/句末标点切块，单块最多 800 rune；同一来源重复添加时重建。
  - BM25 检索（k1=1.2、b=0.75），返回 TopK 片段与 `source#seq`。
  - 索引 `.geekagent/rag/index.json`；提供 `Add` / `Search` / `Stats` / `Lines` / `Clear`。
- `internal/tools`
  - `rag_add`：`path` 必填，经 PathGuard 后采集；输出来源/段数/跳过数，不输出全文。
  - `rag_search`：`query` 必填、`top_k` 可选；输出 `[source#seq] 片段`。
  - `rag_search` 加入只读工具集合；`rag_add` 不加入。
- `internal/permission`
  - 默认 allow 增加 `rag_add` / `rag_search`。
- `cmd/hi-agent`
  - `/rag` 查看索引；`/rag clear` 确认后删除索引。
- 文档、单测、桌面构建与 OpenSpec 勾选。

## Non-goals

- 不做 embedding、向量库、外部检索服务或增量索引。
- 不自动把 RAG 结果注入每一轮；检索由模型显式调用 `rag_search`。
- 不直接采集 URL，不做网页抓取（Day 12 `fetch` 负责抓取）。
- 不做单段编辑/删除；只支持来源重建和整库清空。

## 验收

- `rag_add` 采集文件/目录后只返回统计，输出中不含采集全文。
- `rag_search` 返回相关片段，带 `来源#段号`，最多 TopK 段。
- 同一来源重复添加不会产生重复段；目录重扫会移除已不存在的旧来源。
- 无索引/索引损坏不阻断对话：无索引返回空，损坏返回明确错误；`/rag` 可报告。
- `rag_search` 在工具集收缩后仍可用；默认权限为 allow。
