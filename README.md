<div align="center">

<img src="icon.png" alt="Otter PPT" width="80"/>

# Otter PPT

### AI-Powered PPT Generation Engine — Building Editable PowerPoint via Tool Calls

English | [中文](./README_CN.md)

</div>

---

## 📖 Introduction

Otter PPT is a **tool-call-based AI Agent** PPT generation engine.

Unlike traditional "one-shot JSON generation" approaches, Otter PPT encapsulates every PowerPoint design operation into **callable tools**, allowing AI to build presentations step by step — just like a human designer:

```
User: "Create a PPT about AI"
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
Presentation Object → PPTX Builder → Editable .pptx File
```

## ✨ Key Features

| Feature | Description |
|---------|-------------|
| 🔧 **30+ Design Tools** | A comprehensive toolset covering all PPT capabilities |
| 🎨 **Themes & Styles** | Color schemes, fonts, gradient backgrounds |
| 📝 **Text & Typography** | Titles, body text, bullet lists, rich text |
| 🖼️ **Visual Elements** | Images, shapes (14 types), tables, charts, connectors |
| ✨ **Animations & Transitions** | Element animations, slide transition effects |
| 📐 **Precise Positioning** | Percentage-based coordinate system, resolution-independent |
| 🏗️ **Native PPTX** | Direct OOXML generation, fully editable |
| 🔤 **Font embedding** | Curated Google Fonts + CJK system fonts, embeddable into PPTX |
| 🌐 **HTTP API** | Built-in Gin server with REST API support |
| 🔌 **Multi-Protocol Integration** | MCP, STDIO JSON-RPC, OpenAPI — works with Claude Code, Cursor, Codex, and custom software |
| 📦 **Python SDK** | Lightweight Python client for automation |
| 🖥️ **CLI Tool** | One-command PPT generation from the terminal |

## 🔌 Integrating with Claude Code, Cursor, Codex & Custom Software

Start the MCP stdio service after building:

```bash
./bin/otter-ppt mcp
```

Configure in any MCP-compatible client:

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

AI invocation is not mandatory: you can configure text/image models for Otter PPT to orchestrate, or have external models generate Presentation JSON or tool calls, then hand them to Otter PPT via `/api/v1/build` or `/api/v1/execute`. Other programs can also use `otter-ppt stdio` (JSON-RPC 2.0), or auto-generate clients via [`openapi.yaml`](./openapi.yaml). The Python client is located at [`sdk/python`](./sdk/python). The service runs as a local Go binary.

Full configuration and protocol details: [`INTEGRATION.md`](./INTEGRATION.md).

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
# Set API Key
export TEXT_MODEL_API_KEY="sk-your-key-here"

# Simple mode: fast, direct agent loop (no vision review)
./bin/otter-ppt gen \
  --topic "Future Trends of Artificial Intelligence" \
  --slides 10 \
  --style "tech, dark theme" \
  --language en \
  --mode simple \
  --output my_presentation.pptx

# Workflow mode: plan → build → AI visual review → refine (default)
./bin/otter-ppt gen \
  --topic "Future Trends of Artificial Intelligence" \
  --slides 10 \
  --mode workflow \
  --output my_presentation.pptx
```

| Mode | Speed | Quality | How It Works |
|------|-------|---------|-------------|
| `simple` | ⚡ Fast (~1 min/slide) | Good | Prompt → Agent tools → Auto-fix → PPTX |
| `workflow` | 🐢 Slower (~3 min/slide) | Best | Plan → Build → **Render → AI Vision Review → Refine** → PPTX |
```

### Start HTTP Server

```bash
export TEXT_MODEL_API_KEY="sk-your-key-here"
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

### Presentation Level
| Tool | Description |
|------|-------------|
| `set_title` | Set presentation title |
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
| `set_bg_color` | Set solid color background |
| `set_bg_gradient` | Set gradient background |
| `set_bg_image` | Set image background |

### Text
| Tool | Description |
|------|-------------|
| `add_title` | Add a title |
| `add_text` | Add a text box |
| `add_bullet_list` | Add a bullet list |

### Visual Elements
| Tool | Description |
|------|-------------|
| `add_image` | Add an image |
| `add_shape` | Add a shape (rectangle / oval / arrow / star, etc. — 14 types) |
| `add_table` | Add a table |
| `add_chart` | Add a native editable DrawingML chart (bar / line / pie / area / doughnut) |
| `add_connector` | Add a connector / arrow |

### Element Operations
| Tool | Description |
|------|-------------|
| `update_text` | Update text content |
| `update_style` | Update font style |
| `update_position` | Update position and size |
| `delete_element` | Delete an element |
| `bring_to_front` | Bring to front |
| `send_to_back` | Send to back |
| `set_rotation` | Set rotation angle |
| `set_opacity` | Set opacity |

### Animation & Transitions
| Tool | Description |
|------|-------------|
| `set_transition` | Set slide transition effect |
| `set_animation` | Set element animation |

### State & Export
| Tool | Description |
|------|-------------|
| `get_state` | Get current state |
| `done` | Finish and export |

## 📐 Coordinate System

All positions use **percentage coordinates** (0–100), relative to slide dimensions:

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

- `x, y`: Top-left corner coordinates
- `w, h`: Width and height
- Recommended content range: `x+w ≤ 95`, `y+h ≤ 92`

## 🏗️ Project Structure

```
otter-ppt/
├── cmd/
│   ├── otter-ppt/
│   │   └── main.go           # CLI entry point (serve / gen)
│   └── fontdl/
│       └── main.go           # Font download tool (Google Fonts + sysfonts)
├── internal/
│   ├── model/                # Data models
│   │   ├── presentation.go   # Presentation, Slide, Element
│   │   ├── slide.go          # ElementType, ShapeType, ChartType enums
│   │   ├── background.go     # Background, Gradient, Transition, Animation
│   │   ├── shape.go          # ShapeData, ChartData, ConnectorData
│   │   └── layout.go         # SlideLayout
│   ├── pptoolkit/            # ★ Core: PPT toolset
│   │   ├── session.go        # Session (thread-safe canvas)
│   │   ├── tools.go          # OpenAI tool definitions (30+)
│   │   ├── handlers.go       # Tool dispatch and execution
│   │   └── schema.go         # JSON Schema build helpers
│   ├── agent/                # AI Agent + Workflow pipeline
│   │   ├── agent.go          # LLM ↔ tool call orchestration
│   │   ├── workflow.go       # Multi-phase: plan → build → review → refine
│   │   ├── planner.go        # Phase 1: LLM design planning
│   │   └── vision.go         # Phase 4: Multimodal visual evaluation
│   ├── builder/              # PPTX builder
│   │   ├── builder.go        # Main entry + helper functions
│   │   ├── content_types.go  # [Content_Types].xml
│   │   ├── presentation.go   # presentation.xml
│   │   ├── font_embed.go     # Font embedding (fntdata parts)
│   │   ├── theme_master.go   # theme + slideMaster + slideLayout
│   │   ├── slide.go          # Slide XML generation
│   │   └── elements.go       # Element XML (text/shape/table/chart/connector)
│   ├── fonts/                # ★ Font management
│   │   ├── registry.go       # Font discovery, metadata, catalog
│   │   ├── ttf.go            # TTF/OTF name table parser
│   │   └── builtin.go        # Built-in font recommendations
│   ├── server/               # HTTP API server
│   │   └── server.go         # Gin routes (incl. font endpoints)
│   ├── ai/                   # Legacy one-shot generation mode
│   │   └── generator.go
│   ├── imageutil/            # Image utilities
│   │   └── image.go
│   ├── renderer/             # Slide rendering for visual review
│   │   └── renderer.go       # LibreOffice → PDF → PNG (3-tier fallback)
├── go.mod
├── Makefile
└── README.md
```

## 🔑 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEXT_MODEL_API_KEY` | LLM API key (required) | Falls back to `OPENAI_API_KEY` |
| `TEXT_MODEL_BASE_URL` | LLM API endpoint | Falls back to `OPENAI_BASE_URL` |
| `TEXT_MODEL_NAME` | Text model name | Falls back to `OPENAI_MODEL`, then `gpt-4o` |
| `IMAGE_MODEL_API_KEY` | Image generation API key (optional) | - |
| `IMAGE_MODEL_BASE_URL` | Image generation endpoint | - |
| `IMAGE_MODEL_NAME` | Image model name | - |

### Compatible LLM Services

Set `OPENAI_BASE_URL` to use other compatible services:

```bash
# DeepSeek
export OPENAI_BASE_URL="https://api.deepseek.com/v1"
export OPENAI_MODEL="deepseek-chat"

# Other compatible services
export OPENAI_BASE_URL="https://your-api.com/v1"
```

## 🔤 Font System

Otter PPT includes a built-in font management system that discovers, catalogues, and embeds fonts into PPTX output.

### Features

- **Font Registry**: Automatically scans `assets/fonts/` for `.ttf`, `.otf`, and `.ttc` files
- **Metadata Parsing**: Extracts font family name, PostScript name, weight, and subfamily from font files
- **PPTX Embedding**: Fonts used in the presentation are embedded into the PPTX for cross-platform consistency
- **CJK Support**: Detects and tags CJK fonts (Microsoft YaHei, SimSun, SimHei, KaiTi)
- **Built-in Catalog**: 35+ recommended fonts (system, Google Fonts, open-source) as a reference
- **Custom Installation**: Upload your own fonts via API or manually drop files in `assets/fonts/`

### Downloading Curated Fonts

The `cmd/fontdl` tool downloads open-source fonts from Google Fonts and copies select Windows system CJK fonts:

```bash
# Download Google Fonts (20 Latin fonts) + Windows system CJK fonts
go run ./cmd/fontdl -dir assets/fonts -sysfonts

# Only Google Fonts
go run ./cmd/fontdl -dir assets/fonts

# Only system fonts (CJK)
go run ./cmd/fontdl -dir assets/fonts -no-gstatic -sysfonts
```

**Downloaded Google Fonts**: Inter, Montserrat, Roboto, Open Sans, Lato, Poppins, Raleway, Playfair Display, Merriweather, Lora, JetBrains Mono, Source Code Pro, Bebas Neue, Oswald, Dancing Script, Caveat

**Copied System Fonts** (Windows only): Microsoft YaHei, SimHei (黑体), SimSun (宋体), KaiTi (楷体)

### Font API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/fonts` | GET | List all installed fonts + built-in catalog |
| `/api/v1/fonts/scan` | POST | Re-scan fonts directory |
| `/api/v1/fonts/install` | POST | Upload and install a font file (`.ttf`, `.otf`, `.ttc`) |

#### Install a Custom Font

```bash
curl -X POST http://localhost:8080/api/v1/fonts/install \
  -F "font=@MyCustomFont.ttf"
```

#### List Available Fonts

```bash
curl http://localhost:8080/api/v1/fonts
```

### Adding Fonts Manually

Simply drop `.ttf`, `.otf`, or `.ttc` files into the `assets/fonts/` directory and re-scan:

```bash
# Copy your font file
cp MyFont.ttf assets/fonts/

# Trigger a re-scan via API
curl -X POST http://localhost:8080/api/v1/fonts/scan
```

### Font Categories

| Category | Examples | Use Case |
|----------|----------|----------|
| **Sans-serif** | Inter, Montserrat, Roboto, Open Sans | Modern UI, tech presentations |
| **Serif** | Playfair Display, Lora, Merriweather | Elegant, editorial, formal |
| **Display** | Bebas Neue, Oswald | Headlines, posters |
| **Script** | Dancing Script, Caveat | Decorative, creative |
| **Mono** | JetBrains Mono, Source Code Pro | Code snippets, technical |
| **CJK** | Microsoft YaHei, SimSun, SimHei, KaiTi | Chinese, Japanese, Korean |

## 🔬 Visual Review Architecture

Otter PPT supports a **multi-phase workflow** (`--mode workflow`) that adds AI visual evaluation after the initial build:

```
Phase 1: PLAN     → LLM creates slide outline + design strategy
Phase 2: GATHER   → Pre-generate images in parallel (optional)
Phase 3: BUILD    → Agent tool-calling loop (same as simple mode)
Phase 4: REVIEW   → Render slides → send to vision AI → get design feedback
Phase 5: REFINE   → Agent fixes issues based on AI feedback
Phase 6: POLISH   → Final auto-fix + layout validation
```

### Rendering Backends (3-tier fallback)

The renderer converts PPTX slides into images for the vision model to evaluate. It automatically selects the best available backend:

| Tier | Backend | Output Quality | Dependencies | When to Use |
|------|---------|---------------|-------------|-------------|
| **1** | LibreOffice headless | ⭐⭐⭐ Perfect (true PPTX render) | `soffice` + `pdftoppm` in PATH | Server/Desktop with LibreOffice |
| **2** | Native Go renderer | ⭐⭐ Good (shapes + text + gradients) | None (uses bundled TTF fonts) | Any environment, zero install |
| **3** | Structural text description | ⭐ Functional (JSON-like element dump) | None | Fallback when image API unsupported |

**Auto-detection**: The renderer probes for `soffice`/`libreoffice` and `pdftoppm` at startup. If found, Tier 1 is used. If not, it falls back to Tier 2 (Go native rendering with `golang.org/x/image/font`). If the vision model rejects images, Tier 3 sends structured text.

#### Installing LibreOffice (optional, for best quality)

<details>
<summary>Windows</summary>

Download from https://www.libreoffice.org/download/ and install. The `soffice.exe` will be auto-detected at:
- `C:\Program Files\LibreOffice\program\soffice.exe`
- `C:\Program Files (x86)\LibreOffice\program\soffice.exe`

For `pdftoppm`, install [Poppler for Windows](https://github.com/oschwartz10612/poppler-windows/releases) and add to PATH.
</details>

<details>
<summary>Linux / Docker</summary>

```bash
# Debian/Ubuntu
apt-get install -y libreoffice poppler-utils

# Alpine
apk add libreoffice poppler-utils

# Docker (add to Dockerfile)
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

### Vision Evaluation Flow

```
PPTX → Best available renderer → Slide images (PNG, base64)
                                        ↓
                              Vision LLM (multimodal)
                                        ↓
                         { design_score, content_score,
                           issues[], suggestions[] }
                                        ↓
                    Agent.Refine() → update_position, update_style, etc.
```

If the vision model's overall score ≥ threshold (default 75), the presentation is accepted. Otherwise, feedback is fed back to the agent for up to `MaxRefineRounds` (default 2) iterations.

## 📄 License

MIT License
