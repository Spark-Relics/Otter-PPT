<div align="center">

# 🦦 Otter PPT

### AI-Powered PPT Generation Engine — Build Editable PowerPoint via Tool Calling

中文 | [English](./README.md)

</div>

---

## 📖 Overview

Otter PPT is a **tool-calling AI Agent** engine for generating PowerPoint presentations.

Unlike traditional "one-shot JSON generation" approaches, Otter PPT encapsulates every PowerPoint design operation into **callable tools**, letting the AI build presentations step-by-step like a human designer:

```
User: "Create a presentation about AI"
  ↓
AI Agent (function calling loop)
  ├── set_theme(primary_color="#1A73E8", ...)
  ├── add_slide(layout="title")           → returns slide_id
  ├── add_title(slide_id, text="The Future of AI")
  ├── set_bg_gradient(slide_id, ...)
  ├── add_slide(layout="title_content")   → returns slide_id
  ├── add_bullet_list(slide_id, items=[...])
  ├── add_chart(slide_id, chart_type="bar", ...)
  ├── add_shape(slide_id, shape_type="rounded_rectangle", ...)
  ├── set_transition(slide_id, type="fade")
  ├── set_animation(slide_id, element_id, type="fly_in")
  └── done()
  ↓
Presentation Object → PPTX Builder → Editable .pptx file
```

## ✨ Key Features

| Feature | Description |
|---------|-------------|
| 🔧 **30+ Design Tools** | AI can call a comprehensive set of tools covering all PPT features |
| 🎨 **Themes & Styles** | Color schemes, fonts, gradient backgrounds |
| 📝 **Text & Typography** | Titles, body text, bullet lists, rich text |
| 🖼️ **Visual Elements** | Images, shapes (14 types), tables, charts, connectors |
| ✨ **Animations & Transitions** | Element animations, slide transitions |
| 📐 **Precise Positioning** | Percentage-based coordinate system, resolution-independent |
| 🏗️ **Native PPTX** | Generates raw OOXML, fully editable in PowerPoint |
| 🌐 **HTTP API** | Built-in Gin server with REST API |
| 🖥️ **CLI Tool** | Generate PPTs from the command line |

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- An OpenAI-compatible API key (supports OpenAI / DeepSeek / other compatible services)

### Installation

```bash
git clone https://github.com/Spark-Relics/Otter-PPT.git
cd Otter-PPT/otter-ppt
make build
```

### CLI Generation

```bash
# Set API key
export OPENAI_API_KEY="sk-your-key-here"

# Generate PPT
./bin/otter-ppt gen \
  --topic "Future Trends of Artificial Intelligence" \
  --slides 10 \
  --style "tech, dark theme" \
  --language en \
  --output my_presentation.pptx
```

### Start HTTP Server

```bash
export OPENAI_API_KEY="sk-your-key-here"
./bin/otter-ppt serve --port 8080
```

API usage examples:

```bash
# Generate PPT
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{"topic":"Artificial Intelligence","slides":8,"language":"en","style":"tech"}'

# Build PPTX from JSON
curl -X POST http://localhost:8080/api/v1/build \
  -H "Content-Type: application/json" \
  -d @presentation.json \
  -o output.pptx

# List available tools
curl http://localhost:8080/api/v1/tools
```

## 🔧 Tool Reference

All tools available to the AI Agent:

### Presentation-Level
| Tool | Description |
|------|-------------|
| `set_title` | Set the presentation title |
| `set_theme` | Set global color scheme and fonts |
| `set_slide_size` | Set slide dimensions (16:9 / 4:3) |

### Slide Operations
| Tool | Description |
|------|-------------|
| `add_slide` | Add a new slide |
| `delete_slide` | Delete a slide |
| `duplicate_slide` | Duplicate a slide |
| `move_slide` | Reorder slides |
| `set_notes` | Set speaker notes |

### Background
| Tool | Description |
|------|-------------|
| `set_bg_color` | Set solid background color |
| `set_bg_gradient` | Set gradient background |
| `set_bg_image` | Set image background |

### Text
| Tool | Description |
|------|-------------|
| `add_title` | Add a title element |
| `add_text` | Add a text box |
| `add_bullet_list` | Add a bullet list |

### Visual Elements
| Tool | Description |
|------|-------------|
| `add_image` | Add an image |
| `add_shape` | Add a shape (rectangle/ellipse/arrow/star, 14 types) |
| `add_table` | Add a data table |
| `add_chart` | Add a chart (bar/line/pie, etc.) |
| `add_connector` | Add a connector line/arrow |

### Element Manipulation
| Tool | Description |
|------|-------------|
| `update_text` | Update text content |
| `update_style` | Update font/style |
| `update_position` | Update position and size |
| `delete_element` | Delete an element |
| `bring_to_front` | Bring to front (z-order) |
| `send_to_back` | Send to back (z-order) |
| `set_rotation` | Set rotation angle |
| `set_opacity` | Set opacity/transparency |

### Animations & Transitions
| Tool | Description |
|------|-------------|
| `set_transition` | Set slide transition effect |
| `set_animation` | Set element animation |

### State & Export
| Tool | Description |
|------|-------------|
| `get_state` | Get current presentation state |
| `done` | Signal completion and export |

## 📐 Coordinate System

All positions use **percentage coordinates** (0-100), relative to slide dimensions:

```
┌──────────────────────────────────┐
│ 0,0                         100,0│
│                                  │
│   ┌─── x,y ───┐                 │
│   │            │                 │
│   │   W × H    │                 │
│   │            │                 │
│   └────────────┘                 │
│                                  │
│ 0,100                      100,100│
└──────────────────────────────────┘
```

- `x, y`: Top-left corner position
- `w, h`: Width and height
- Recommended bounds: `x+w ≤ 95`, `y+h ≤ 92`

## 🏗️ Project Structure

```
otter-ppt/
├── cmd/
│   └── otter-ppt/
│       └── main.go           # CLI entry (serve / gen)
├── internal/
│   ├── model/                # Data models
│   │   ├── presentation.go   # Presentation, Slide, Element
│   │   ├── slide.go          # ElementType, ShapeType, ChartType enums
│   │   ├── background.go     # Background, Gradient, Transition, Animation
│   │   ├── shape.go          # ShapeData, ChartData, ConnectorData
│   │   └── layout.go         # SlideLayout
│   ├── pptoolkit/            # ★ Core: PPT tool set
│   │   ├── session.go        # Session (thread-safe canvas)
│   │   ├── tools.go          # OpenAI tool definitions (30+)
│   │   ├── handlers.go       # Tool dispatch & execution
│   │   └── schema.go         # JSON Schema builder helpers
│   ├── agent/                # AI Agent loop
│   │   └── agent.go          # LLM ↔ tool-calling orchestration
│   ├── builder/              # PPTX builder
│   │   ├── builder.go        # Main entry + helpers
│   │   ├── content_types.go  # [Content_Types].xml
│   │   ├── presentation.go   # presentation.xml
│   │   ├── theme_master.go   # theme + slideMaster + slideLayout
│   │   ├── slide.go          # Slide XML generation
│   │   └── elements.go       # Element XML (text/shape/table/chart/connector)
│   ├── server/               # HTTP API server
│   │   └── server.go         # Gin routes
│   ├── ai/                   # Legacy one-shot generation mode
│   │   └── generator.go
│   └── imageutil/            # Image utilities
│       └── image.go
├── go.mod
├── Makefile
└── README.md
```

## 🔑 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OPENAI_API_KEY` | API key (required) | - |
| `OPENAI_BASE_URL` | Custom API endpoint | OpenAI official |
| `OPENAI_MODEL` | Model name | `gpt-4o` |

### Compatible LLM Services

Set `OPENAI_BASE_URL` to use other compatible services:

```bash
# DeepSeek
export OPENAI_BASE_URL="https://api.deepseek.com/v1"
export OPENAI_MODEL="deepseek-chat"

# Other compatible services
export OPENAI_BASE_URL="https://your-api.com/v1"
```

## 📄 License

MIT License
