# Proposal: Day5 历史压缩

## Intent

对话变长时把旧消息压成一条摘要，保留最近 6 条，控制上下文体积。

## Scope

- 自动压缩（超阈值）+ `/compact`
- `GEEKAGENT_MAX_HISTORY` 可调阈值

## Non-goals

- 二次摘要保护、轮内中途压缩、多会话
