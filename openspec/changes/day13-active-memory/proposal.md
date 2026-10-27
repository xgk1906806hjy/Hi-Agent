# Change: Day13 active memory recall

## Intent

把已有的项目长期记忆从「等模型主动调用 `memory_search`」升级为每轮自动召回：对用户输入做 BM25 检索，取最相关的 Top5 记忆块，以临时 system 消息注入本轮请求。用户不需要先说「回忆一下」，模型也能看到相关偏好/事实。

## Scope

- `internal/memory`
  - 记忆内容切块：按段落/句子聚合，单块最多约 240 个 rune；超长句子硬切。
  - 轻量分词：拉丁字母/数字按词；CJK 文档侧单字 + 双字 gram，查询侧主要用双字 gram（单字查询保留单字），统一小写。
  - BM25（k1=1.2、b=0.75）对块打分，返回 Top5；仅保留至少命中一个查询词项的块，同分时优先较新的记忆。
  - 新增 `Recall(query, topN)` 与 `RecallNote(query)`；`RecallNote` 生成可直接放进 system 的说明文本。
- `internal/chat`
  - 新增 `SetRecall(func(query string) string)` 钩子；`StreamReply` 在用户入队、自动压缩之后调用，生成本轮临时 `recallNote`。
  - `outbound` 把 `recallNote` 排在 `AGENTS.md`、技能片段之后，history 之前；不写入持久 history。
  - 本轮结束/中断/回滚时清空 `recallNote`，避免泄漏到下一轮。
  - 命中时输出进度行 `[主动记忆：已注入相关记忆]`。
  - `Delegate` 子对话继承 recall 钩子；子任务输入也按同一规则召回。
- `cmd/hi-agent`
  - 启动时注入 `chat.SetRecall(memory.RecallNote)`；`/project` 换目录后随相对路径自然切换记忆文件。
- 文档、单测与 OpenSpec 勾选。

## Non-goals

- 不做 embedding、向量库或外部检索服务。
- 不建持久化 BM25 索引；每次按当前 `memory.json` 计算，文件规模按项目长期记忆控制。
- 不改变 `memory_search` 的显式子串搜索语义。
- 不做记忆修改/删除、自动写入记忆或 Day 14 的 RAG 文档索引。

## 验收

- 存在相关记忆时，不提「回忆」也能在发给模型的 system 中看到 Top5 记忆块。
- 无关输入不注入记忆；没有 `memory.json` 或文件损坏时对话继续，不阻断回复。
- 长记忆被切成多个块，单块不超过约 240 字；Top5 数量稳定。
- 显式 `memory_search` 行为保持原样。
