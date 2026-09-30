# Proposal: Day4 文件工具与注册表

## Intent

把读写从 shell 拆出：只读免确认，写入先 diff 再确认；工具统一 `Register`。

## Scope

- `Register` 注册表
- `ls` / `glob` / `read` / `write` / `patch`

## Non-goals

- 目录隔离、undo、权限策略表
