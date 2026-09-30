# OpenSpec 使用说明（Hi-agent）

本项目使用 [OpenSpec](https://openspec.dev/) 做 **规格驱动开发**：先约定要建什么，再让 Agent 按 tasks 实现。

## 1. 目录结构

```
openspec/
├── specs/              # 已落地的系统行为（真相）
│   └── <domain>/spec.md
├── changes/            # 进行中的变更（一文件夹一事）
│   └── <change-name>/
│       ├── proposal.md
│       ├── design.md
│       ├── tasks.md
│       └── specs/      # 相对主规格的 delta
└── config.yaml         # 可选项目配置
```

## 2. 终端 vs 聊天

| 在哪里 | 命令 | 用途 |
|--------|------|------|
| 终端 | `openspec init` / `update` / `list` / `show` / `validate` / `view` | CLI 管理 |
| Cursor 聊天 | `/opsx-propose`、`/opsx-apply`、`/opsx-archive` 等 | 与 AI 协作写规格与实现 |

本仓库已用 `openspec init --tools cursor` 生成命令，**Cursor 聊天**里使用：

| 命令 | 作用 |
|------|------|
| `/opsx-explore` | 探索想法（不落盘） |
| `/opsx-propose` | 创建 change：proposal / specs / design / tasks |
| `/opsx-apply` | 按 tasks 实现 |
| `/opsx-update` | 修订进行中的变更产物 |
| `/opsx-sync` | 把 delta specs 合并进主 specs |
| `/opsx-archive` | 归档已完成变更 |

重启或重载 Cursor 后命令才会出现在斜杠菜单。

## 3. 推荐工作流

```text
/opsx-explore     （可选：想清楚范围）
      ↓
/opsx-propose     生成 proposal / specs / design / tasks
      ↓
人工审阅 tasks 与场景
      ↓
/opsx-apply       按 tasks 实现
      ↓
/opsx-archive     合并 specs，归档 change
```

当前已创建首个变更：`openspec/changes/day01-repl/`（Day 1 REPL）。
下一步在聊天执行：`/opsx-apply`（或指定 day01-repl）。

## 4. 与 Day 方案配合

- 路线图见 [开发方案-按Day.md](./开发方案-按Day.md)
- 每个 Day（或每周阶段）对应一个 `openspec/changes/<name>/`
- 阶段完成后 archive，主 `openspec/specs/` 反映当前 Agent 能力

## 5. 常用 CLI

```bash
openspec list
openspec show <change-name>
openspec validate <change-name>
openspec view
openspec update          # 升级 CLI 后刷新本仓库生成的 skills/commands
```

## 6. 升级

```bash
npm install -g @fission-ai/openspec@latest
cd <项目根>
openspec update
```
