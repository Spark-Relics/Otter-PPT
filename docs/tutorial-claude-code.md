# Claude Code 使用教程

通过 MCP 协议将 Otter PPT 接入 Claude Code，让 Claude 直接为你生成 PPT。

## 前置条件

- 已安装 [Claude Code](https://docs.anthropic.com/en/docs/claude-code)
- 已编译 Otter PPT 二进制文件

## 第 1 步：编译

```bash
git clone https://github.com/Spark-Relics/Otter-PPT.git
cd Otter-PPT/otter-ppt
make build
```

编译后二进制位于 `bin/otter-ppt`（Windows 为 `bin/otter-ppt.exe`）。

## 第 2 步：配置 MCP

在项目根目录或全局创建 `.claude/settings.json`：

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

> Windows 用户请使用完整路径，例如 `"C:\\Project\\otter-ppt\\bin\\otter-ppt.exe"`。

## 第 3 步：启动 Claude Code

```bash
claude
```

Claude Code 启动时会自动加载 MCP 服务。你可以在对话中直接让 Claude 创建 PPT：

```
帮我用 otter-ppt 做一个 8 页的 AI 趋势报告 PPT，科技蓝色主题
```

Claude 会自动调用以下工具链：

```
set_theme → add_slide → add_title → add_bullet_list → add_chart → set_transition → export_pptx
```

## 第 4 步：获取生成的文件

Claude 调用 `export_pptx` 时会指定输出路径。你可以在对话中指定：

```
把 PPT 导出到 ~/Desktop/ai-trends.pptx
```

## 支持的工具列表

| 工具 | 说明 |
|------|------|
| `set_title` / `set_theme` | 设置标题和主题 |
| `add_slide` | 添加幻灯片（title / title_content / two_column / section / blank） |
| `add_title` / `add_text` / `add_bullet_list` | 添加文本元素 |
| `add_image` | 添加图片（支持 image_prompt 让 AI 生图） |
| `add_shape` | 添加形状（14 种） |
| `add_table` | 添加表格（支持合并单元格） |
| `add_chart` | 添加图表（13 种，含 3D） |
| `add_connector` | 添加连接线 |
| `set_transition` | 设置幻灯片转场动画 |
| `set_animation` | 设置元素动画 |
| `set_bg_color` / `set_bg_gradient` / `set_bg_image` | 设置背景 |
| `set_notes` | 设置演讲者备注 |
| `update_text` / `update_style` / `update_position` | 修改元素 |
| `delete_element` / `delete_slide` | 删除元素或幻灯片 |
| `bring_to_front` / `send_to_back` | 调整层级 |
| `set_rotation` / `set_opacity` | 旋转和透明度 |
| `move_slide` / `duplicate_slide` | 幻灯片操作 |
| `export_pptx` | 导出为 .pptx 文件 |
| `reset_session` | 重置会话 |

## 常见问题

### Q: Claude 提示找不到 MCP 服务？

确认二进制路径正确，且文件有执行权限：
```bash
chmod +x /absolute/path/to/otter-ppt
```

### Q: 如何使用 AI 自动生成图片？

配置图像模型环境变量：
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

### Q: 可以不用 AI 模型吗？

可以。MCP 工具调用本身不需要 API Key，Claude Code 自带模型能力。Otter PPT 仅充当工具执行器。
