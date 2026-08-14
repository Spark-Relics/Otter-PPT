# Cursor 使用教程

通过 MCP 协议将 Otter PPT 接入 Cursor IDE，在编辑器中直接生成 PPT。

## 前置条件

- 已安装 [Cursor](https://cursor.sh/)
- 已编译 Otter PPT 二进制文件

## 第 1 步：编译

```bash
git clone https://github.com/Spark-Relics/Otter-PPT.git
cd Otter-PPT/otter-ppt
make build
```

## 第 2 步：配置 MCP

打开 Cursor → **Settings** → **Features** → **MCP**，点击 **Add new MCP Server**：

| 字段 | 值 |
|------|-----|
| Name | `otter-ppt` |
| Type | `stdio` |
| Command | `/absolute/path/to/otter-ppt` |
| Args | `mcp` |

或者直接编辑项目根目录 `.cursor/mcp.json`：

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

## 第 3 步：在 Cursor Chat 中使用

打开 Cursor 的 Chat 面板（`Cmd/Ctrl + L`），切换到 **Agent** 模式：

```
帮我生成一个产品发布会 PPT，10 页，深色科技风，要有柱状图和饼图
```

Cursor 会自动调用 Otter PPT 的 MCP 工具，逐步构建幻灯片并导出。

## 第 4 步：交互式编辑

你也可以先生成再修改：

```
帮我用 otter-ppt：
1. 创建一个空白 PPT
2. 第 1 页用 title 布局，标题是「2024 年度报告」
3. 第 2 页用 title_content，加一个柱状图展示季度收入
4. 给所有页面加 fade 转场
5. 导出到 ./output/report.pptx
```

## 配置环境变量（可选）

如果需要 AI 自动生成图片，在 `.cursor/mcp.json` 中添加 `env`：

```json
{
  "mcpServers": {
    "otter-ppt": {
      "command": "/path/to/otter-ppt",
      "args": ["mcp"],
      "env": {
        "IMAGE_MODEL_API_KEY": "sk-your-key",
        "IMAGE_MODEL_BASE_URL": "https://api.openai.com/v1",
        "IMAGE_MODEL_NAME": "dall-e-3"
      }
    }
  }
}
```

## 验证连接

配置完成后，在 Cursor Chat 中输入：

```
列出 otter-ppt 可用的工具
```

Cursor 会调用 `tools.list` 并返回所有可用工具。
