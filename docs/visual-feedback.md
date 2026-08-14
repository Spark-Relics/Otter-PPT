# 视觉反馈循环：让外部 AI 看到渲染效果

## 问题

以前，视觉反馈只在 `otter-ppt gen --mode workflow` 内部闭环。外部 AI（Claude Code、Cursor、Codex）通过 MCP/REST 接入时，看不到渲染效果，无法利用自己的视觉能力做优化。

## 解决方案：`render_slides` 工具 + `/api/v1/render` 端点

现在，渲染能力已暴露为标准工具和 API，外部 AI 可以：

```
构建幻灯片 → 调用 render_slides → 获得渲染图片 → 视觉分析 → 修复问题 → 再次 render_slides 验证
```

---

## 三层渲染支持

| 层级 | 后端 | 质量 | 依赖 |
|------|------|------|------|
| **Tier 1** | LibreOffice → PDF → PNG | ⭐⭐⭐ 完美 | `soffice` + `pdftoppm` |
| **Tier 2** | Go 原生渲染（形状+文字+渐变） | ⭐⭐ 良好 | 无 |
| **Tier 3** | 结构化文本描述 | ⭐ 可用 | 无 |

自动降级：优先 LibreOffice，不可用时用 Go 原生，再不行用文本描述。

---

## MCP 模式（Claude Code / Cursor）

`render_slides` 工具已注册到 MCP 工具列表中。调用后返回 **MCP image content blocks**，多模态 AI（如 Claude）可以直接"看到"图片。

### 工作流示例

在 Claude Code 中：

```
用户：帮我做一个 3 页的 AI 趋势 PPT

Claude：
  1. set_theme(primary_color="#1A73E8", ...)
  2. add_slide(layout="title")
  3. add_title(slide_id="slide-1", text="AI Trends 2024")
  4. add_slide(layout="title_content")
  5. add_bullet_list(slide_id="slide-2", items=["LLM", "Multimodal", "Agents"])
  6. add_slide(layout="section")
  7. add_title(slide_id="slide-3", text="Thank You")
  
  8. render_slides()  ← 关键！Claude 获得渲染图片

  [Claude 看到图片后]
  "我发现第 2 页的文字框和标题靠得太近了，让我调整一下"
  
  9. update_position(slide_id="slide-2", element_id="elem-3", y=35)
  10. render_slides()  ← 再次验证
  
  [Claude 确认效果好了]
  11. export_pptx(output_path="ai-trends.pptx")
```

### MCP 返回格式

```json
{
  "content": [
    {"type": "text", "text": "Rendered 3 slides using libreoffice backend..."},
    {"type": "image", "data": "iVBORw0KGgo...", "mimeType": "image/png"},
    {"type": "text", "text": "Slide 1: [描述]..."},
    {"type": "image", "data": "iVBORw0KGgo...", "mimeType": "image/png"},
    {"type": "text", "text": "Slide 2: [描述]..."},
    ...
  ]
}
```

> Claude 会自动识别 `type: "image"` 的内容块并展示为图片，然后用自己的视觉能力分析。

---

## REST API 模式

### `POST /api/v1/render`

传入 Presentation JSON，返回每页的渲染图片（base64 PNG）。

```bash
# 先构建幻灯片
RESP=$(curl -s -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "calls": [
      {"name": "set_theme", "arguments": {"primary_color": "#1A73E8"}},
      {"name": "add_slide", "arguments": {"layout": "title"}},
      {"name": "add_title", "arguments": {"slide_id": "slide-1", "text": "Hello", "x": 10, "y": 35, "w": 80, "h": 20}}
    ]
  }')

# 提取 presentation JSON 并渲染
echo "$RESP" | jq '.presentation' | curl -s -X POST http://localhost:8080/api/v1/render \
  -H "Content-Type: application/json" \
  -d @- | jq '.slides[0] | {slide_num, backend}'
```

**响应格式：**

```json
{
  "backend": "libreoffice",
  "slide_count": 3,
  "slides": [
    {
      "slide_num": 1,
      "width": 1920,
      "height": 1080,
      "image_base64": "iVBORw0KGgo...",
      "description": "Slide 1 [title]:\n  Background: solid #1A73E8\n  [title] pos(10,35)-(90,55) text=\"Hello\""
    }
  ],
  "hint": "Use the image_base64 fields to visually evaluate..."
}
```

### 外部 AI 工作流（Python 示例）

```python
import requests
import base64
import json

BASE = "http://localhost:8080"

# 1. 构建幻灯片
resp = requests.post(f"{BASE}/api/v1/execute", json={
    "calls": [
        {"name": "set_theme", "arguments": {"primary_color": "#1A73E8"}},
        {"name": "add_slide", "arguments": {"layout": "title"}},
        {"name": "add_title", "arguments": {
            "slide_id": "slide-1", "text": "AI Trends",
            "x": 10, "y": 35, "w": 80, "h": 20
        }},
    ]
})
presentation = resp.json()["presentation"]

# 2. 渲染获取图片
resp = requests.post(f"{BASE}/api/v1/render", json=presentation)
slides = resp.json()["slides"]

# 3. 把图片发给你的 AI 模型做视觉评审
for slide in slides:
    # 解码图片用于本地 AI 分析
    img_data = base64.b64decode(slide["image_base64"])
    # 或者直接把 base64 发给你的多模态 LLM
    review = your_vision_llm.analyze(img_data, prompt="检查设计问题")
    
    print(f"Slide {slide['slide_num']}: {review}")
    
    # 4. 根据评审结果修复
    fixes = parse_fixes(review)
    if fixes:
        resp = requests.post(f"{BASE}/api/v1/execute", json={
            "presentation": presentation,
            "calls": fixes
        })
        presentation = resp.json()["presentation"]

# 5. 最终导出
resp = requests.post(f"{BASE}/api/v1/build", json=presentation)
with open("output.pptx", "wb") as f:
    f.write(resp.content)
```

---

## STDIO JSON-RPC 模式

```json
{"jsonrpc":"2.0","id":1,"method":"tools.call","params":{"name":"render_slides","arguments":{}}}
```

返回幻灯片图片 base64 + 描述。

---

## 安装 LibreOffice（获得最佳渲染质量）

### Windows
下载 https://www.libreoffice.org/download/，安装后 `soffice.exe` 会被自动检测。

### Linux / Docker
```bash
apt-get install -y libreoffice poppler-utils
```

### macOS
```bash
brew install --cask libreoffice
brew install poppler
```

---

## 总结

| 接入方式 | 渲染工具 | AI 视觉能力 |
|----------|---------|------------|
| MCP (Claude Code) | `render_slides` | Claude 原生多模态，直接看图分析 |
| MCP (Cursor) | `render_slides` | 取决于 Cursor 配置的模型 |
| REST API | `POST /api/v1/render` | 你的代码控制，发任意视觉 LLM |
| STDIO JSON-RPC | `render_slides` | 自定义编排器处理 |
| CLI `gen --mode workflow` | 自动内建 | 内置 GPT-4o 视觉评审 |
