# REST API 使用教程

通过 HTTP 接口调用 Otter PPT，适用于任何编程语言和自动化场景。

## 启动服务

```bash
# 设置 API Key（使用 /generate 时必需）
export TEXT_MODEL_API_KEY="sk-your-key"
export TEXT_MODEL_BASE_URL="https://api.openai.com/v1"
export TEXT_MODEL_NAME="gpt-4o"

# 启动服务
./bin/otter-ppt serve --port 8080
```

## API 端点总览

| 方法 | 路径 | 说明 | 需要 API Key |
|------|------|------|:---:|
| `POST` | `/api/v1/generate` | AI 自动生成 PPT | ✅ |
| `POST` | `/api/v1/execute` | 执行工具调用（无状态） | ❌ |
| `POST` | `/api/v1/build` | 从 JSON 构建 PPTX | ❌ |
| `POST` | `/api/v1/render` | 渲染幻灯片为图片（视觉反馈） | ❌ |
| `GET` | `/api/v1/tools` | 列出所有工具定义 | ❌ |
| `GET` | `/api/v1/download` | 下载生成的文件 | ❌ |
| `GET` | `/health` | 健康检查 | ❌ |

---

## 1. AI 自动生成

```bash
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "人工智能行业趋势",
    "slides": 10,
    "language": "zh",
    "style": "tech, dark theme"
  }'
```

**响应：**

```json
{
  "task_id": "abc-123",
  "status": "completed",
  "download_url": "/api/v1/download?id=abc-123"
}
```

**下载文件：**

```bash
curl -o output.pptx http://localhost:8080/api/v1/download?id=abc-123
```

### Simple vs Workflow 模式

```bash
# Simple 模式：快速直接生成
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "产品介绍",
    "slides": 8,
    "mode": "simple"
  }'

# Workflow 模式：规划→构建→渲染→AI视觉评审→修复（默认，质量更高）
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "产品介绍",
    "slides": 8,
    "mode": "workflow"
  }'
```

---

## 2. 从 JSON 构建（无需 AI）

```bash
curl -X POST http://localhost:8080/api/v1/build \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Presentation",
    "slides": [
      {
        "id": "s1",
        "layout": "title",
        "elements": [
          {
            "id": "t1",
            "type": "title",
            "text": "Hello World",
            "rect": {"x": 10, "y": 35, "w": 80, "h": 20}
          }
        ]
      }
    ]
  }' \
  -o output.pptx
```

---

## 3. 工具调用执行

无需 AI 模型，直接发送工具调用序列：

```bash
# 第一批调用
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "calls": [
      {"name": "set_title", "arguments": {"title": "Demo"}},
      {"name": "add_slide", "arguments": {"layout": "title"}},
      {"name": "add_slide", "arguments": {"layout": "title_content"}}
    ]
  }'
```

**响应：**

```json
{
  "presentation": {
    "title": "Demo",
    "slides": [...]
  }
}
```

继续编辑时传回上一次的 `presentation`：

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "calls": [
      {"name": "add_title", "arguments": {"slide_id": "slide-1", "text": "标题"}}
    ],
    "presentation": {"title": "Demo", "slides": [...]}
  }'
```

最后构建为 PPTX：

```bash
curl -X POST http://localhost:8080/api/v1/build \
  -H "Content-Type: application/json" \
  -d @presentation.json \
  -o final.pptx
```

---

## 4. 查看可用工具

```bash
curl http://localhost:8080/api/v1/tools | python -m json.tool
```

---

## 5. 健康检查

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## 6. 渲染幻灯片（视觉反馈）

将 Presentation JSON 渲染为图片，让 AI 视觉评审后优化：

```bash
curl -X POST http://localhost:8080/api/v1/render \
  -H "Content-Type: application/json" \
  -d @presentation.json
```

**响应：**

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
      "description": "Slide 1 [title]: ..."
    }
  ]
}
```

详见 [视觉反馈教程](./visual-feedback.md)。

---

## OpenAPI / Swagger

完整的 API 定义见 [`openapi.yaml`](../openapi.yaml)。你可以用任何 OpenAPI 3 生成器创建客户端：

```bash
# 生成 TypeScript 客户端
npx @openapitools/openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-fetch \
  -o ./generated-client

# 生成 Java 客户端
openapi-generator-cli generate \
  -i openapi.yaml \
  -g java \
  -o ./java-client
```

---

## Docker 部署

```bash
# 构建镜像
docker build -t otter-ppt .

# 运行
docker run -p 8080:8080 \
  -e TEXT_MODEL_API_KEY="sk-your-key" \
  -e TEXT_MODEL_BASE_URL="https://api.openai.com/v1" \
  otter-ppt serve --port 8080
```
