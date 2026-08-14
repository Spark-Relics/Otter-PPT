# CLI 命令行使用教程

通过命令行直接生成 PPT，无需编程。

## 编译

```bash
git clone https://github.com/Spark-Relics/Otter-PPT.git
cd Otter-PPT/otter-ppt
make build
```

编译后二进制位于 `bin/otter-ppt`。

## 命令总览

```bash
./bin/otter-ppt --help
```

| 子命令 | 说明 |
|--------|------|
| `gen` | AI 自动生成 PPT |
| `serve` | 启动 HTTP API 服务 |
| `mcp` | 启动 MCP stdio 服务 |
| `stdio` | 启动 JSON-RPC stdio 服务 |

---

## gen — 生成 PPT

### 基本用法

```bash
export TEXT_MODEL_API_KEY="sk-your-key"

./bin/otter-ppt gen \
  --topic "人工智能行业趋势" \
  --slides 10 \
  --style "tech, dark theme" \
  --language zh \
  --output ai-trends.pptx
```

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--topic` | （必填） | PPT 主题 |
| `--slides` | `8` | 幻灯片数量 |
| `--style` | `""` | 风格描述（如 "tech, minimalist"） |
| `--language` | `en` | 语言（`en` / `zh`） |
| `--mode` | `workflow` | 生成模式：`simple`（快速）或 `workflow`（最佳质量） |
| `--output` | `output.pptx` | 输出文件路径 |

### Simple 模式（快速）

```bash
./bin/otter-ppt gen \
  --topic "产品介绍" \
  --slides 8 \
  --mode simple \
  --output product.pptx
```

速度：约 1 分钟/页。流程：Prompt → Agent 工具调用 → 自动修复 → PPTX。

### Workflow 模式（最佳质量，默认）

```bash
./bin/otter-ppt gen \
  --topic "产品介绍" \
  --slides 8 \
  --mode workflow \
  --output product.pptx
```

速度：约 3 分钟/页。流程：规划 → 构建 → 渲染截图 → AI 视觉评审 → 修复 → PPTX。

### 模式对比

| 模式 | 速度 | 质量 | 流程 |
|------|------|------|------|
| `simple` | ⚡ ~1 分钟/页 | 良好 | Prompt → Agent → Auto-fix → PPTX |
| `workflow` | 🐢 ~3 分钟/页 | 最佳 | Plan → Build → **Render → Vision Review → Refine** → PPTX |

---

## serve — HTTP 服务

```bash
export TEXT_MODEL_API_KEY="sk-your-key"

./bin/otter-ppt serve --port 8080
```

启动后可通过 REST API 调用，详见 [REST API 教程](./tutorial-rest-api.md)。

---

## mcp — MCP 服务

```bash
./bin/otter-ppt mcp
```

作为 MCP stdio 服务运行，供 Claude Code / Cursor 等客户端调用。详见 [Claude Code 教程](./tutorial-claude-code.md)。

---

## stdio — JSON-RPC 服务

```bash
./bin/otter-ppt stdio
```

作为 JSON-RPC 2.0 stdio 服务运行，供自定义程序调用。详见 [Codex / STDIO 教程](./tutorial-codex.md)。

---

## 环境变量

| 变量 | 说明 | 必填 |
|------|------|:----:|
| `TEXT_MODEL_API_KEY` | LLM API Key | `gen` 和 `serve /generate` 时必填 |
| `TEXT_MODEL_BASE_URL` | LLM API 地址 | 否（默认 OpenAI） |
| `TEXT_MODEL_NAME` | LLM 模型名 | 否 |
| `IMAGE_MODEL_API_KEY` | 图像生成 API Key | 否 |
| `IMAGE_MODEL_BASE_URL` | 图像 API 地址 | 否 |
| `IMAGE_MODEL_NAME` | 图像模型名 | 否 |

> `OPENAI_API_KEY` / `OPENAI_BASE_URL` 作为 `TEXT_MODEL_*` 的兼容别名仍然支持。

---

## 配合 AI 图片生成

```bash
export TEXT_MODEL_API_KEY="sk-your-key"
export IMAGE_MODEL_API_KEY="sk-your-key"
export IMAGE_MODEL_BASE_URL="https://api.openai.com/v1"
export IMAGE_MODEL_NAME="dall-e-3"

./bin/otter-ppt gen \
  --topic "世界名画赏析" \
  --slides 10 \
  --mode workflow \
  --output art.pptx
```

当 Agent 调用 `add_image` 并提供 `image_prompt` 时，会自动调用图像模型生成图片。
