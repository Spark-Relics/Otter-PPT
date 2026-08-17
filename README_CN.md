<div align="center">

<img src="icon.png" alt="Otter PPT" width="80"/>

# Otter PPT

**AI 驱动的演示文稿生成器 — 通过工具调用 Agent 构建 PPTX**

[English](./README.md) | [中文](./README_CN.md)

[![CI](https://github.com/Spark-Relics/Otter-PPT/actions/workflows/ci.yml/badge.svg)](https://github.com/Spark-Relics/Otter-PPT/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

## 项目简介

Otter PPT 是一个基于 Go 语言的服务，使用 **AI Agent 工具调用** 来生成完全可编辑的 `.pptx` 文件。与传统的「让 LLM 一次性吐出一大坨 JSON」不同，Otter PPT 将 PowerPoint 中的每个设计操作封装成独立的工具（就像 PPT 软件里的 UI 操作），AI 逐步调用这些工具来搭建幻灯片，拥有与人类设计师一样的精细控制力。

### 工作原理

```
用户输入: "做一个关于AI的PPT"
      │
      ▼
┌─────────────────────────────────┐
│       AI Agent（LLM 循环）       │
│      function-calling × N 轮     │
├─────────────────────────────────┤
│ set_theme(...)                  │  ← 设置主题配色
│ add_slide(layout="title")       │  ← 添加幻灯片
│ add_title(text="AI的未来")      │  ← 添加标题
│ set_bg_gradient(...)            │  ← 设置渐变背景
│ add_shape(...)                  │  ← 添加形状
│ add_chart(...)                  │  ← 添加图表
│ set_transition(type="fade")     │  ← 设置切换动画
│ done()                          │  ← 完成
└──────────────┬──────────────────┘
               │
               ▼
┌─────────────────────────────────┐
│     PPTX 构建器（OOXML）         │
│    结构化模型 → .pptx 文件       │
└─────────────────────────────────┘
```

### 核心设计决策

| 决策 | 原因 |
|---|---|
| **工具调用 Agent**（非一次性 JSON） | AI 像人类设计师一样逐步操作，可以随时添加、移动、修改元素 |
| **原生 OOXML 生成**（不依赖第三方 PPT 库） | 完全控制 PPTX 的每个特性：渐变、形状、表格、切换动画 |
| **百分比坐标系**（0-100） | 分辨率无关的布局，适配任何幻灯片尺寸 |
| **结构化 JSON 中间层** | 演示文稿状态可检查、可调试、可序列化 |

---

## 功能特性

### 🎨 设计工具（30+ 个）

AI Agent 可以使用以下所有操作：

| 分类 | 工具 |
|---|---|
| **演示文稿** | `set_title`（标题）、`set_theme`（主题）、`set_slide_size`（尺寸） |
| **幻灯片** | `add_slide`（添加）、`delete_slide`（删除）、`duplicate_slide`（复制）、`move_slide`（移动）、`set_notes`（备注） |
| **背景** | `set_bg_color`（纯色）、`set_bg_gradient`（渐变）、`set_bg_image`（图片） |
| **文本** | `add_title`（标题）、`add_text`（正文）、`add_bullet_list`（项目符号列表） |
| **视觉元素** | `add_image`（图片）、`add_shape`（形状）、`add_table`（表格）、`add_chart`（图表）、`add_connector`（连接线） |
| **元素操作** | `update_text`（改文本）、`update_style`（改样式）、`update_position`（改位置）、`delete_element`（删除） |
| **特效** | `set_transition`（切换）、`set_animation`（动画）、`set_rotation`（旋转）、`set_opacity`（透明度） |
| **层级** | `bring_to_front`（置顶）、`send_to_back`（置底）、`group_elements`（组合） |
| **控制** | `get_state`（查看状态）、`done`（完成） |

### 支持的元素类型

- **文本**：标题、正文、项目符号列表（支持自定义符号）
- **形状**：矩形、圆角矩形、椭圆、三角形、菱形、箭头、五边形、六边形、星形、标注、线条、心形、云形
- **表格**：支持表头样式、交替行颜色、自定义配色
- **原生可编辑图表**：柱状图、条形图、折线图、饼图、面积图、圆环图（DrawingML 图表数据缓存，可在 PowerPoint 中编辑样式与数据点）
- **连接线**：直线和箭头
- **背景**：纯色、线性/径向渐变

### 幻灯片特性

- 16:9 和 4:3 幻灯片尺寸
- 幻灯片切换效果（淡入、推入、擦除、分割、覆盖、缩放、变体）
- 演讲者备注
- Z 轴层级管理
- 元素旋转和透明度

---

## 多平台与第三方集成

Otter PPT 将演示文稿模型、工具执行器、AI 提供方和传输协议分层。AI 调用不是强制的：可以为服务配置文字/图像模型，也可以由外部模型生成完整 Presentation JSON 或工具调用，再交给 Otter PPT 执行和渲染。

| 接入方式 | 适用场景 |
|---|---|
| **MCP stdio** | Claude Code、Cursor 及其他 MCP 客户端 |
| **STDIO JSON-RPC 2.0** | Codex 包装器、编辑器插件、桌面软件和自研编排器 |
| **REST + OpenAPI 3.0** | 跨语言服务、Web 应用、自动生成客户端 |
| **Python SDK** | Python 自动化与 AI 工作流 |

启动 MCP 服务：

```bash
./bin/otter-ppt mcp
```

MCP 客户端配置示例：

```json
{
  "mcpServers": {
    "otter-ppt": {
      "command": "/absolute/path/to/otter-ppt",
      "args": ["mcp"]
    }
  }
}
```

不支持 MCP 时可运行 `./bin/otter-ppt stdio`，或使用 [`openapi.yaml`](./openapi.yaml) 和 [`sdk/python`](./sdk/python)。完整说明见 [`INTEGRATION.md`](./INTEGRATION.md)。

### 📚 各平台使用教程

| 平台 / 工具 | 教程链接 |
|-------------|----------|
| 命令行（gen / serve / mcp） | [CLI 教程](./docs/tutorial-cli.md) |
| Claude Code（MCP） | [Claude Code 教程](./docs/tutorial-claude-code.md) |
| Cursor（MCP） | [Cursor 教程](./docs/tutorial-cursor.md) |
| Codex / STDIO JSON-RPC | [Codex / STDIO 教程](./docs/tutorial-codex.md) |
| Python SDK | [Python SDK 教程](./docs/tutorial-python-sdk.md) |
| REST API（curl / OpenAPI） | [REST API 教程](./docs/tutorial-rest-api.md) |
| Docker 部署 | [Docker 教程](./docs/tutorial-docker.md) |

---

## 快速开始

### 前置条件

- **Go 1.22+**
- OpenAI 兼容的 API Key（OpenAI、DeepSeek、Moonshot 等）

### 安装

```bash
git clone https://github.com/Spark-Relics/Otter-PPT.git
cd Otter-PPT/otter-ppt
go mod tidy
go build -o bin/otter-ppt ./cmd/otter-ppt
```

### 生成模式

| 模式 | 速度 | 质量 | 工作流程 |
|------|------|------|---------|
| `simple` | ⚡ 快（~1 分钟/页） | 良好 | 提示词 → Agent 工具调用 → 自动修复 → PPTX |
| `workflow`（默认） | 🐢 较慢（~3 分钟/页） | 最佳 | 规划 → 构建 → **渲染截图 → AI 视觉评审 → 修复** → PPTX |

### 命令行使用

```bash
# 设置环境变量
export TEXT_MODEL_API_KEY="sk-..."
export TEXT_MODEL_BASE_URL="https://api.openai.com/v1"  # 可选，用于自定义端点
export TEXT_MODEL_NAME="gpt-4o"                         # 可选，默认 gpt-4o

# Simple 模式：快速生成，不走视觉评审
./bin/otter-ppt gen \
  --topic "人工智能的发展趋势" \
  --slides 10 \
  --style "科技感、深色主题" \
  --language zh \
  --mode simple \
  --output 我的演示文稿.pptx

# Workflow 模式：规划 → 构建 → AI 视觉评审 → 修复（默认）
./bin/otter-ppt gen \
  --topic "人工智能的发展趋势" \
  --slides 10 \
  --mode workflow \
  --output 我的演示文稿.pptx
```

### HTTP 服务模式

```bash
export TEXT_MODEL_API_KEY="sk-..."
./bin/otter-ppt serve --port 8080
```

**API 端点：**

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/v1/generate` | 根据主题生成 PPTX |
| `POST` | `/api/v1/build` | 从演示文稿 JSON 构建 PPTX |
| `GET` | `/api/v1/tools` | 列出所有可用工具 |
| `GET` | `/health` | 健康检查 |

**示例：**

```bash
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{"topic":"AI Future","slides":8,"language":"en","style":"tech"}'
```

### Makefile 命令

```bash
make build    # 编译二进制
make run      # 直接运行服务
make serve    # 编译 + 启动服务
make test     # 运行测试
make tidy     # 整理依赖
make clean    # 清理构建产物
```

---

## 项目结构

```
otter-ppt/
├── cmd/
│   └── otter-ppt/
│       └── main.go              # CLI 入口（serve / gen 命令）
├── internal/
│   ├── model/                   # 核心数据模型
│   │   ├── presentation.go      # Presentation、Slide、Element、Theme
│   │   ├── slide.go             # 元素类型、形状/图表/动画枚举
│   │   ├── background.go        # Background、Gradient、Transition、Animation
│   │   ├── shape.go             # ShapeData、ChartData、ConnectorData
│   │   └── layout.go            # 幻灯片布局类型
│   ├── pptoolkit/               # ★ 核心创新：工具调用层
│   │   ├── session.go           # 会话状态（"画布"）
│   │   ├── tools.go             # OpenAI 工具定义（30+ 个）
│   │   ├── handlers.go          # 工具分发 + map→struct 转换
│   │   └── schema.go            # JSON schema 构建辅助
│   ├── agent/                   # AI Agent + 多阶段工作流
│   │   ├── agent.go             # LLM ↔ 工具调用编排
│   │   ├── workflow.go          # 多阶段：规划 → 构建 → 评审 → 修复
│   │   ├── planner.go           # 第 1 阶段：LLM 设计规划
│   │   └── vision.go            # 第 4 阶段：多模态视觉评审
│   ├── builder/                 # PPTX 渲染器（原生 OOXML）
│   │   ├── builder.go           # ZIP 包写入器 + 辅助函数
│   │   ├── content_types.go     # [Content_Types].xml + .rels
│   │   ├── presentation.go      # presentation.xml + rels
│   │   ├── theme_master.go      # theme、slideMaster、slideLayout
│   │   ├── slide.go             # slide XML（背景、切换）
│   │   └── elements.go          # 元素渲染器（文本、形状、表格...）
│   ├── server/                  # HTTP API（Gin）
│   │   └── server.go
│   ├── ai/                      # 旧版一次性生成器（已弃用）
│   │   └── generator.go
│   ├── imageutil/               # 图片工具
│   │   └── image.go
│   ├── renderer/                # 幻灯片渲染（视觉评审用）
│   │   └── renderer.go          # LibreOffice → PDF → PNG（三级降级）
├── go.mod
└── Makefile
```

---

## 架构详解

### Agent 循环

```go
for step := 0; step < maxSteps; step++ {
    // 1. 将对话历史 + 工具列表发给 LLM
    resp := client.CreateChatResponse(messages, tools)

    // 2. 如果 LLM 返回工具调用，执行它们
    for _, toolCall := range resp.ToolCalls {
        result := session.ExecuteTool(toolCall.Name, toolCall.Args)
        messages = append(messages, toolResultMessage(result))
    }

    // 3. 如果 LLM 调用 "done" 或停止，则完成
    if toolName == "done" { break }
}

// 4. 将最终状态渲染为 PPTX
builder.New(session.Presentation()).Save("output.pptx")
```

### 坐标系统

所有元素位置使用百分比（0–100）相对于幻灯片尺寸：

```
(0,0) ────────────────────── (100,0)
  │         x            w      │
  │      ┌────────────────┐    │
  │   y  │                │    │
  │      │     元素       │    │
  │   h  │                │    │
  │      └────────────────┘    │
(0,100) ────────────────────── (100,100)
```

这使得布局与分辨率无关——同一个模型适用于 16:9、4:3 或任何自定义尺寸。

---

## 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `TEXT_MODEL_API_KEY` | 回退到 `OPENAI_API_KEY` | LLM API 密钥（必填） |
| `TEXT_MODEL_BASE_URL` | 回退到 `OPENAI_BASE_URL` | LLM API 端点 |
| `TEXT_MODEL_NAME` | 回退到 `OPENAI_MODEL`，再回退 `gpt-4o` | 文本模型名称 |
| `IMAGE_MODEL_API_KEY` | — | 图片生成 API 密钥（可选） |
| `IMAGE_MODEL_BASE_URL` | — | 图片生成端点 |
| `IMAGE_MODEL_NAME` | — | 图片模型名称 |

### 自定义 LLM 服务商

Otter PPT 兼容任何 OpenAI 格式的 API：

```bash
# DeepSeek
export OPENAI_BASE_URL="https://api.deepseek.com/v1"
export OPENAI_MODEL="deepseek-chat"

# Moonshot (Kimi)
export OPENAI_BASE_URL="https://api.moonshot.cn/v1"
export OPENAI_MODEL="moonshot-v1-32k"
```

---

## 开发路线

- [x] 真实图片嵌入（PNG/JPEG/GIF/SVG，本地与 data: URI）
- [x] 原生图表 XML（13 种图表，含 3D、趋势线、误差线、次坐标轴）
- [x] 演讲者备注（notesSlide XML 部件）
- [x] 动画 XML 渲染（7 种动画 × 触发方式 × 方向）
- [x] SVG → 原生 PPTX 编译（import_svg，freeform custGeom）
- [x] 智能布局自动排列（validate_layout / auto_fix_layout / apply_smart_layout，18 种智能模板）
- [x] 模板系统（load_template：从现有 .pptx 导入色板/字体/尺寸/版式清单）
- [x] Web UI 实时预览（/preview/:token 查看页 + 轮询自动刷新 + 工具调用推送）

---

## 🔬 视觉评审架构

Otter PPT 支持**多阶段工作流**（`--mode workflow`），在初始构建后加入 AI 视觉评审环节：

```
第 1 阶段: 规划 (PLAN)     → LLM 创建幻灯片大纲 + 设计策略
第 2 阶段: 收集 (GATHER)   → 并行预生成所需图片（可选）
第 3 阶段: 构建 (BUILD)    → Agent 工具调用循环（与 simple 模式相同）
第 4 阶段: 评审 (REVIEW)   → 渲染幻灯片 → 发送给视觉 AI → 获取设计反馈
第 5 阶段: 修复 (REFINE)   → Agent 根据反馈修复问题
第 6 阶段: 抛光 (POLISH)   → 最终自动修复 + 布局校验
```

### 渲染后端（三级降级）

渲染器将 PPTX 幻灯片转换为图片，供视觉模型评审。自动选择最佳可用后端：

| 层级 | 后端 | 输出质量 | 依赖 | 适用场景 |
|------|------|---------|------|---------|
| **1** | LibreOffice 无头模式 | ⭐⭐⭐ 完美（真实 PPTX 渲染） | `soffice` + `pdftoppm` 在 PATH 中 | 已安装 LibreOffice 的服务器/桌面 |
| **2** | Go 原生渲染器 | ⭐⭐ 良好（形状 + 文字 + 渐变） | 无（使用内置 TTF 字体） | 任何环境，零安装 |
| **3** | 结构化文本描述 | ⭐ 可用（类 JSON 元素信息） | 无 | 视觉模型不支持图片时的兜底 |

**自动检测**：渲染器在启动时探测 `soffice`/`libreoffice` 和 `pdftoppm`。找到则使用 Tier 1；否则降级到 Tier 2（Go 原生渲染，使用 `golang.org/x/image/font`）；如果视觉模型不支持图片输入，则降级到 Tier 3 发送结构化文本。

#### 安装 LibreOffice（可选，获得最佳质量）

<details>
<summary>Windows</summary>

从 https://www.libreoffice.org/download/ 下载安装。`soffice.exe` 会被自动检测：
- `C:\Program Files\LibreOffice\program\soffice.exe`
- `C:\Program Files (x86)\LibreOffice\program\soffice.exe`

`pdftoppm` 需安装 [Poppler for Windows](https://github.com/oschwartz10612/poppler-windows/releases) 并加入 PATH。
</details>

<details>
<summary>Linux / Docker</summary>

```bash
# Debian/Ubuntu
apt-get install -y libreoffice poppler-utils

# Alpine
apk add libreoffice poppler-utils

# Docker（加入 Dockerfile）
RUN apt-get update && apt-get install -y libreoffice poppler-utils && rm -rf /var/lib/apt/lists/*
```
</details>

<details>
<summary>macOS</summary>

```bash
brew install --cask libreoffice
brew install poppler
```
</details>

### 视觉评审流程

```
PPTX → 最佳可用渲染器 → 幻灯片图片（PNG, base64）
                              ↓
                    视觉 LLM（多模态）
                              ↓
             { design_score, content_score,
               issues[], suggestions[] }
                              ↓
          Agent.Refine() → update_position, update_style 等
```

如果视觉模型总分 ≥ 阈值（默认 75），则接受演示文稿。否则将反馈送回 Agent，最多迭代 `MaxRefineRounds` 轮（默认 2 轮）。

---

## 许可证

MIT License

## 贡献

欢迎提交 PR！重大变更请先开 Issue 讨论。
