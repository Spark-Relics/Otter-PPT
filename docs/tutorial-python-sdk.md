# Python SDK 使用教程

通过 Python 客户端调用 Otter PPT 服务，实现 PPT 自动化生成。

## 安装

### 方式一：从源码安装

```bash
cd otter-ppt/sdk/python
pip install -e .
```

### 方式二：直接安装

```bash
pip install ./sdk/python
```

## 快速开始

### 1. 启动服务

```bash
# 先启动 Otter PPT HTTP 服务
./bin/otter-ppt serve --port 8080
```

### 2. AI 自动生成

```python
from otter_ppt import OtterPPT

client = OtterPPT("http://localhost:8080")

# AI 自动生成 PPT
result = client.generate(
    topic="2024 AI 行业趋势报告",
    slides=10,
    language="zh",
    style="tech, dark theme"
)

print(f"任务 ID: {result['task_id']}")
print(f"下载地址: {result['download_url']}")

# 下载 PPTX 文件
client.download(result["download_url"], "ai-report.pptx")
client.close()
```

### 3. 从 JSON 构建（无需 AI）

```python
from otter_ppt import OtterPPT

client = OtterPPT("http://localhost:8080")

# 手动构建 Presentation JSON
presentation = {
    "title": "季度报告",
    "slides": [
        {
            "id": "s1",
            "layout": "title",
            "elements": [
                {
                    "id": "t1",
                    "type": "title",
                    "text": "2024 Q3 季度报告",
                    "rect": {"x": 10, "y": 35, "w": 80, "h": 20}
                }
            ]
        },
        {
            "id": "s2",
            "layout": "title_content",
            "elements": [
                {
                    "id": "t2",
                    "type": "title",
                    "text": "营收概览",
                    "rect": {"x": 10, "y": 10, "w": 80, "h": 15}
                },
                {
                    "id": "c2",
                    "type": "chart",
                    "rect": {"x": 10, "y": 30, "w": 80, "h": 60},
                    "chart": {
                        "chart_type": "column",
                        "title": "月度营收（万元）",
                        "categories": ["7月", "8月", "9月"],
                        "series": [
                            {"name": "2024", "values": [120, 150, 180], "color": "#1A73E8"}
                        ],
                        "show_legend": True
                    }
                }
            ]
        }
    ]
}

# 构建 PPTX
client.build(presentation, "quarterly-report.pptx")
client.close()
print("✅ quarterly-report.pptx 生成完毕")
```

### 4. 使用工具调用（无状态模式）

```python
import requests

BASE = "http://localhost:8080"

# 累积式工具调用
calls = [
    {"name": "set_title", "arguments": {"title": "Product Launch"}},
    {"name": "add_slide", "arguments": {"layout": "title"}},
    {"name": "add_slide", "arguments": {"layout": "title_content"}},
]

resp = requests.post(f"{BASE}/api/v1/execute", json={
    "calls": calls,
    "presentation": None  # 第一次调用传 null
})

data = resp.json()
presentation = data["presentation"]

# 继续追加操作
more_calls = [
    {"name": "add_title", "arguments": {"slide_id": "slide-1", "text": "Product Launch"}},
    {"name": "add_bullet_list", "arguments": {
        "slide_id": "slide-2",
        "items": ["核心功能", "技术架构", "市场策略"],
        "x": 10, "y": 30, "w": 80, "h": 50
    }},
]

resp = requests.post(f"{BASE}/api/v1/execute", json={
    "calls": more_calls,
    "presentation": presentation
})

# 构建最终文件
final_pres = resp.json()["presentation"]
resp = requests.post(f"{BASE}/api/v1/build", json=final_pres)

with open("product.pptx", "wb") as f:
    f.write(resp.content)
```

## API 速查

| 方法 | 说明 |
|------|------|
| `client.generate(topic, slides, language, style)` | AI 自动生成 PPT |
| `client.build(presentation_dict, output_path)` | 从 JSON 构建 PPTX |
| `client.download(url, output_path)` | 下载生成的 PPTX |
| `client.list_tools()` | 列出所有可用工具 |
| `client.close()` | 关闭连接 |

## 环境变量

服务端需要在启动时配置（Python SDK 无需关心）：

| 变量 | 说明 |
|------|------|
| `TEXT_MODEL_API_KEY` | LLM API Key |
| `TEXT_MODEL_BASE_URL` | LLM API 地址 |
| `TEXT_MODEL_NAME` | LLM 模型名 |
| `IMAGE_MODEL_API_KEY` | 图像生成 API Key（可选） |

## 完整示例：批量生成

```python
from otter_ppt import OtterPPT
import time

client = OtterPPT("http://localhost:8080")

topics = [
    {"topic": "人工智能趋势", "slides": 8, "style": "tech"},
    {"topic": "营销策略分析", "slides": 12, "style": "business"},
    {"topic": "产品设计复盘", "slides": 10, "style": "creative"},
]

for i, t in enumerate(topics):
    print(f"[{i+1}/{len(topics)}] 正在生成: {t['topic']}...")
    result = client.generate(
        topic=t["topic"],
        slides=t["slides"],
        language="zh",
        style=t["style"]
    )
    filename = f"output_{i+1}.pptx"
    client.download(result["download_url"], filename)
    print(f"  ✅ 已保存: {filename}")

client.close()
```
