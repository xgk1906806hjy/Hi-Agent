# Design: Day1 REPL 地基

## 说明

Day1 是**能力里程碑**，不是目录名。代码落在统一工程：

```
cmd/hi-agent/main.go
internal/config/
internal/color/
internal/chat/
```

## 技术选型

| 项 | 选择 |
|----|------|
| 语言 | Go |
| 模型 | `sashabaranov/go-openai` + 自定义 BaseURL |
| REPL | `bufio.Scanner` |
| 配置 | `joho/godotenv` |
| TTY | `golang.org/x/term` |

## 关键决策

1. **history = messages**：后续工具、system、压缩只追加不同 role。
2. **出错回滚**：请求失败时去掉刚入队的 user。
3. **颜色集中**：业务只调 `color` 包。
4. **单线程 REPL**：读完再处理。
