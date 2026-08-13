<div align="center">

# 🦦 Otter PPT

### AI 驱动的 PPT 生成引擎 — 用工具调用构建可编辑的 PowerPoint

[English](./README_EN.md) | 中文

</div>

---

## 📖 项目简介

Otter PPT 是一个**基于 AI Agent + 工具调用**的 PPT 生成引擎。

与传统的"一次性生成 JSON"方案不同，Otter PPT 将 PowerPoint 的每一步设计操作封装为**可调用的工具**，让 AI 像人类设计师一样逐步搭建 PPT：

```
用户: "做一个关于AI的PPT"
  ↓
AI Agent (function calling 循环)
  ├── set_theme(primary_color="#1A73E8", ...)
  ├── add_slide(layout="title")           → 返回 slide_id
  ├── add_title(slide_id, text="AI的未来")
  ├── set_bg_gradient(slide_id, ...)
  ├── add_slide(layout="title_content")   → 返回 slide_id
  ├── add_bullet_list(slide_id, items=[...])
  ├── add_chart(slide_id, chart_type="bar", ...)
  ├── add_shape(slide_id, shape_type="rounded_rectangle", ...)
  ├── set_transition(slide_id, type="fade")
  ├── set_animation(slide_id, element_id, type="fly_in")
  └── done()
  ↓
Presentation 对象 → PPTX Builder → 可编辑的 .pptx 文件
```

## ✨ 核心特性

| 特性 | 说明 |
|------|------|
| 🔧 **30+ 设计工具** | AI 可以调用覆盖 PPT 所有功能的工具集 |
| 🎨 **主题与样式** | 颜色方案、字体、渐变背景 |
| 📝 **文本与排版** | 标题、正文、项目符号列表、富文本 |
| 🖼️ **视觉元素** | 图片、形状（14种）、表格、图表、连接线 |
| ✨ **动画与切换** | 元素动画、幻灯片切换效果 |
| 📐 **精确定位** | 百分比坐标系，分辨率无关 |
| 🏗️ **原生 PPTX** | 直接生成 OOXML，完全可编辑 |
| 🌐 **HTTP API** | 内置 Gin 服务器，支持 REST API 调用 |
| 🖥️ **CLI 工具** | 命令行一键生成 PPT |

## 🚀 快速开始

### 环境要求

- Go 1.22+
- OpenAI 兼容的 API Key（支持 OpenAI / DeepSeek / 其他兼容服务）

### 安装

```bash
git clone https://github.com/Spark-Relics/Otter-PPT.git
cd Otter-PPT/otter-ppt
make build
```

### 命令行生成

```bash
# 设置 API Key
export OPENAI_API_KEY="sk-your-key-here"

# 生成 PPT
./bin/otter-ppt gen \
  --topic "人工智能的未来发展趋势" \
  --slides 10 \
  --style "科技感、深色主题" \
  --language zh \
  --output my_presentation.pptx
```

### 启动 HTTP 服务

```bash
export OPENAI_API_KEY="sk-your-key-here"
./bin/otter-ppt serve --port 8080
```

API 调用示例：

```bash
# 生成 PPT
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{"topic":"人工智能","slides":8,"language":"zh","style":"科技感"}'

# 从 JSON 构建 PPTX
curl -X POST http://localhost:8080/api/v1/build \
  -H "Content-Type: application/json" \
  -d @presentation.json \
  -o output.pptx

# 查看可用工具
curl http://localhost:8080/api/v1/tools
```

## 🔧 工具列表

AI Agent 可调用的全部工具：

### 演示文稿级别
| 工具 | 说明 |
|------|------|
| `set_title` | 设置演示文稿标题 |
| `set_theme` | 设置全局颜色方案和字体 |
| `set_slide_size` | 设置幻灯片尺寸（16:9 / 4:3） |

### 幻灯片操作
| 工具 | 说明 |
|------|------|
| `add_slide` | 添加新幻灯片 |
| `delete_slide` | 删除幻灯片 |
| `duplicate_slide` | 复制幻灯片 |
| `move_slide` | 调整幻灯片顺序 |
| `set_notes` | 设置演讲者备注 |

### 背景
| 工具 | 说明 |
|------|------|
| `set_bg_color` | 设置纯色背景 |
| `set_bg_gradient` | 设置渐变背景 |
| `set_bg_image` | 设置图片背景 |

### 文本
| 工具 | 说明 |
|------|------|
| `add_title` | 添加标题 |
| `add_text` | 添加文本框 |
| `add_bullet_list` | 添加项目符号列表 |

### 视觉元素
| 工具 | 说明 |
|------|------|
| `add_image` | 添加图片 |
| `add_shape` | 添加形状（矩形/椭圆/箭头/星形等14种） |
| `add_table` | 添加表格 |
| `add_chart` | 添加图表（柱状/折线/饼图等） |
| `add_connector` | 添加连接线/箭头 |

### 元素操作
| 工具 | 说明 |
|------|------|
| `update_text` | 更新文本内容 |
| `update_style` | 更新字体样式 |
| `update_position` | 更新位置和大小 |
| `delete_element` | 删除元素 |
| `bring_to_front` | 置于顶层 |
| `send_to_back` | 置于底层 |
| `set_rotation` | 设置旋转角度 |
| `set_opacity` | 设置透明度 |

### 动画与切换
| 工具 | 说明 |
|------|------|
| `set_transition` | 设置幻灯片切换效果 |
| `set_animation` | 设置元素动画 |

### 状态与导出
| 工具 | 说明 |
|------|------|
| `get_state` | 获取当前状态 |
| `done` | 完成并导出 |

## 📐 坐标系统

所有位置使用**百分比坐标**（0-100），相对于幻灯片尺寸：

```
┌─────────────────────────────────┐
│ 0,0                        100,0│
│                                 │
│   ┌─── x,y ───┐                │
│   │            │                │
│   │   W × H    │                │
│   │            │                │
│   └────────────┘                │
│                                 │
│ 0,100                     100,100│
└─────────────────────────────────┘
```

- `x, y`: 左上角坐标
- `w, h`: 宽度和高度
- 建议内容范围：`x+w ≤ 95`, `y+h ≤ 92`

## 🏗️ 项目结构

```
otter-ppt/
├── cmd/
│   └── otter-ppt/
│       └── main.go           # CLI 入口 (serve / gen)
├── internal/
│   ├── model/                # 数据模型
│   │   ├── presentation.go   # Presentation, Slide, Element
│   │   ├── slide.go          # ElementType, ShapeType, ChartType 枚举
│   │   ├── background.go     # Background, Gradient, Transition, Animation
│   │   ├── shape.go          # ShapeData, ChartData, ConnectorData
│   │   └── layout.go         # SlideLayout
│   ├── pptoolkit/            # ★ 核心：PPT 工具集
│   │   ├── session.go        # Session (线程安全的画布)
│   │   ├── tools.go          # OpenAI 工具定义 (30+)
│   │   ├── handlers.go       # 工具调度与执行
│   │   └── schema.go         # JSON Schema 构建辅助
│   ├── agent/                # AI Agent 循环
│   │   └── agent.go          # LLM ↔ 工具调用编排
│   ├── builder/              # PPTX 构建器
│   │   ├── builder.go        # 主入口 + 辅助函数
│   │   ├── content_types.go  # [Content_Types].xml
│   │   ├── presentation.go   # presentation.xml
│   │   ├── theme_master.go   # theme + slideMaster + slideLayout
│   │   ├── slide.go          # 幻灯片 XML 生成
│   │   └── elements.go       # 元素 XML (文本/形状/表格/图表/连接线)
│   ├── server/               # HTTP API 服务
│   │   └── server.go         # Gin 路由
│   ├── ai/                   # 传统单次生成模式
│   │   └── generator.go
│   └── imageutil/            # 图片处理工具
│       └── image.go
├── go.mod
├── Makefile
└── README.md
```

## 🔑 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `OPENAI_API_KEY` | API 密钥（必填） | - |
| `OPENAI_BASE_URL` | 自定义 API 地址 | OpenAI 官方 |
| `OPENAI_MODEL` | 模型名称 | `gpt-4o` |

### 兼容的 LLM 服务

设置 `OPENAI_BASE_URL` 即可使用其他兼容服务：

```bash
# DeepSeek
export OPENAI_BASE_URL="https://api.deepseek.com/v1"
export OPENAI_MODEL="deepseek-chat"

# 其他兼容服务
export OPENAI_BASE_URL="https://your-api.com/v1"
```

## 📄 许可证

MIT License

