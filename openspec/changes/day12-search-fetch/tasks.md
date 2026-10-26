# Tasks

- [x] `internal/tools/search.go`：注册 `search`，字面/正则、路径范围、噪声目录与二进制过滤、200 条上限
- [x] `internal/tools/fetch.go`：注册 `fetch`，确认、http/https、超时与大小上限、HTML 转纯文本
- [x] `tools` 注册顺序与只读集合加入 `search`
- [x] `permission` 默认 allow/ask 与 `Subject`（search→pattern、fetch→url）
- [x] 单测：search / fetch / permission
- [x] 文档：tools、permission、README、路线图；OpenSpec 勾选
