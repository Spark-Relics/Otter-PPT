<div align="center">

# 🦦 Otter PPT

**AI 驱动的演示文稿生成器 — 通过工具调用 Agent 构建 PPTX**

[English](./README.md) | [中文](./README_CN.md)

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

### 命令行使用

```bash
# 设置环境变量
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选，用于自定义端点
export OPENAI_MODEL="gpt-4o"                         # 可选，默认 gpt-4o

# 生成演示文稿
./bin/otter-ppt gen \
  --topic "人工智能的发展趋势" \
  --slides 10 \
  --style "科技感、深色主题" \
  --language zh \
  --output 我的演示文稿.pptx
```

### HTTP 服务模式

```bash
export OPENAI_API_KEY="sk-..."

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
│   ├── agent/                   # AI Agent 循环
│   │   └── agent.go             # LLM ↔ 工具调用编排
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
│   └── imageutil/               # 图片工具
│       └── image.go
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
| `OPENAI_API_KEY` | *（必填）* | LLM API 密钥 |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | 自定义端点（DeepSeek、Moonshot 等） |
| `OPENAI_MODEL` | `gpt-4o` | 模型名称 |

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

- [ ] 真实图片嵌入（目前渲染为占位符）
- [ ] 原生图表 XML（目前为数据摘要卡片）
- [ ] 演讲者备注（notesSlide XML 部件）
- [ ] 动画 XML 渲染
- [ ] 智能布局自动排列
- [ ] 模板系统
- [ ] Web UI 实时预览

---

## 许可证

MIT License

## 贡献

欢迎提交 PR！重大变更请先开 Issue 讨论。
