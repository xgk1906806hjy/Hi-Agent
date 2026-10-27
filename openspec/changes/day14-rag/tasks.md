# Tasks

- [x] `internal/rag`：索引读写、文件/目录采集、切块、BM25、`Add` / `Search` / `Stats` / `Lines` / `Clear`
- [x] `internal/rag` 单测：来源段号、重复添加、目录跳过、长文本切块、空/损坏索引
- [x] `internal/tools`：`rag_add` / `rag_search` 注册、PathGuard、只读集合
- [x] `internal/tools` 单测：采集不返回全文、检索带来源段号、TopK
- [x] `internal/permission`：默认 allow 与测试
- [x] `cmd/hi-agent`：`/rag`、`/rag clear`、帮助文案
- [x] 文档：rag、tools、repl、permission、README、路线图；OpenSpec 勾选
- [x] 构建 GUI 并更新 `dist/Hi-agent.exe` 与桌面 exe
