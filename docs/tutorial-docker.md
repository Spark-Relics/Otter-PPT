# Docker 部署教程

将 Otter PPT 容器化部署，适用于生产环境和 CI/CD。

## Dockerfile

在项目根目录创建 `Dockerfile`：

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o otter-ppt ./cmd/otter-ppt

FROM alpine:3.20

RUN apk add --no-cache ca-certificates libreoffice-fonts
COPY --from=builder /app/otter-ppt /usr/local/bin/otter-ppt

EXPOSE 8080
ENTRYPOINT ["otter-ppt"]
CMD ["serve", "--port", "8080"]
```

## 构建镜像

```bash
docker build -t otter-ppt:latest .
```

## 运行容器

### 基本启动

```bash
docker run -d \
  --name otter-ppt \
  -p 8080:8080 \
  -e TEXT_MODEL_API_KEY="sk-your-key" \
  -e TEXT_MODEL_BASE_URL="https://api.openai.com/v1" \
  -e TEXT_MODEL_NAME="gpt-4o" \
  otter-ppt:latest
```

### 使用配置文件

```bash
docker run -d \
  --name otter-ppt \
  -p 8080:8080 \
  --env-file .env \
  otter-ppt:latest
```

`.env` 文件：

```env
TEXT_MODEL_API_KEY=sk-your-key
TEXT_MODEL_BASE_URL=https://api.openai.com/v1
TEXT_MODEL_NAME=gpt-4o
IMAGE_MODEL_API_KEY=sk-your-key
IMAGE_MODEL_BASE_URL=https://api.openai.com/v1
IMAGE_MODEL_NAME=dall-e-3
```

### 挂载输出目录

```bash
docker run -d \
  --name otter-ppt \
  -p 8080:8080 \
  -v /path/to/output:/app/output \
  --env-file .env \
  otter-ppt:latest
```

## Docker Compose

```yaml
version: "3.8"

services:
  otter-ppt:
    build: .
    image: otter-ppt:latest
    ports:
      - "8080:8080"
    environment:
      - TEXT_MODEL_API_KEY=${TEXT_MODEL_API_KEY}
      - TEXT_MODEL_BASE_URL=${TEXT_MODEL_BASE_URL}
      - TEXT_MODEL_NAME=${TEXT_MODEL_NAME}
    volumes:
      - ./output:/app/output
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

启动：

```bash
docker compose up -d
```

## 验证

```bash
# 健康检查
curl http://localhost:8080/health

# 列出工具
curl http://localhost:8080/api/v1/tools | python -m json.tool

# 生成 PPT
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{"topic":"AI Trends","slides":5,"language":"en"}'
```

## 多阶段构建优化

最终镜像仅约 20MB（Alpine + 二进制），无 Go 编译工具链残留。

## 注意事项

- LibreOffice 字体包用于渲染截图功能（workflow 模式），如不需要可移除 `libreoffice-fonts`
- 容器内默认端口 `8080`
- 输出文件路径默认 `/app/output`
