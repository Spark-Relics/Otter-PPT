# 📚 Otter PPT 使用教程

根据你的使用场景选择对应教程：

## 快速入口

| 你是… | 推荐教程 | 说明 |
|--------|----------|------|
| **命令行用户** | [CLI 教程](./tutorial-cli.md) | 一行命令生成 PPT |
| **Claude Code 用户** | [Claude Code 教程](./tutorial-claude-code.md) | MCP 集成 |
| **Cursor 用户** | [Cursor 教程](./tutorial-cursor.md) | IDE 内 MCP |
| **Codex / 自定义客户端** | [Codex / STDIO 教程](./tutorial-codex.md) | JSON-RPC 通用集成 |
| **Python 开发者** | [Python SDK 教程](./tutorial-python-sdk.md) | pip install 自动化 |
| **REST API 调用方** | [REST API 教程](./tutorial-rest-api.md) | 任意语言 HTTP |
| **运维 / DevOps** | [Docker 教程](./tutorial-docker.md) | 容器化部署 |

## 集成方式概览

```
┌──────────────────────────────────────────────────────────┐
│                   你的应用 / AI 客户端                      │
├──────────┬──────────┬──────────┬──────────┬───────────────┤
│  Claude  │  Cursor  │  Codex   │  Python  │  REST / curl  │
│  Code    │          │  / STDIO │  SDK     │               │
│ (MCP)    │ (MCP)    │ (JSON-   │ (HTTP)   │ (HTTP)        │
│          │          │  RPC)    │          │               │
├──────────┴──────────┴──────────┴──────────┴───────────────┤
│                    Otter PPT 服务                          │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  30+ 工具（PPT 设计操作）                             │  │
│  │  Presentation Model → PPTX Builder → .pptx          │  │
│  └─────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

## 所有教程文件

1. **[CLI 教程](./tutorial-cli.md)** — 命令行 `gen` / `serve` / `mcp` / `stdio`
2. **[Claude Code 教程](./tutorial-claude-code.md)** — MCP 配置 + 实战
3. **[Cursor 教程](./tutorial-cursor.md)** — IDE MCP 配置
4. **[Codex / STDIO 教程](./tutorial-codex.md)** — JSON-RPC + Python/Node 示例
5. **[Python SDK 教程](./tutorial-python-sdk.md)** — `pip install` + 完整代码示例
6. **[REST API 教程](./tutorial-rest-api.md)** — curl + OpenAPI 客户端生成
7. **[Docker 教程](./tutorial-docker.md)** — Dockerfile + Compose 部署

## 快速决策树

```
想用 Otter PPT？
├── 只是快速生成一份 PPT
│   └── → CLI: ./otter-ppt gen --topic "..." --output out.pptx
├── 在 AI 编辑器里交互式生成
│   ├── Claude Code → MCP 配置
│   └── Cursor → MCP 配置
├── 用代码自动化
│   ├── Python → pip install sdk/python
│   └── 其他语言 → REST API / OpenAPI
├── 自定义 AI 编排器
│   └── STDIO JSON-RPC 或 REST /execute
└── 生产环境部署
    └── Docker / Docker Compose
```
