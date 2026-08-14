# Codex / Windsurf / 其他 MCP 客户端使用教程

适用于任何支持 MCP 协议的客户端（Codex、Windsurf、Continue 等）以及 JSON-RPC STDIO 模式。

## MCP 模式

### 配置

在任何支持 MCP 的客户端中注册 Otter PPT：

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

### 使用

客户端会自动发现 Otter PPT 的工具集。直接在对话中请求生成 PPT 即可。

---

## STDIO JSON-RPC 模式

如果不使用 MCP，可以通过通用 JSON-RPC 2.0 协议通信。

### 启动

```bash
otter-ppt stdio
```

进程会从 stdin 读取换行分隔的 JSON-RPC 消息，向 stdout 写入响应。

### 支持的方法

| 方法 | 说明 |
|------|------|
| `tools.list` | 列出所有可用工具 |
| `tools.call` | 调用一个工具 |
| `presentation.get` | 获取当前 Presentation JSON |
| `session.reset` | 重置会话 |

### 消息格式

**请求示例：**

```json
{"jsonrpc":"2.0","id":1,"method":"tools.call","params":{"name":"set_title","arguments":{"title":"AI Trends"}}}
```

```json
{"jsonrpc":"2.0","id":2,"method":"tools.call","params":{"name":"add_slide","arguments":{"layout":"title"}}}
```

```json
{"jsonrpc":"2.0","id":3,"method":"presentation.get"}
```

**响应格式：**

```json
{"jsonrpc":"2.0","id":1,"result":{"success":true,"slide_id":"slide-1"}}
```

### Python 示例

```python
import subprocess
import json

proc = subprocess.Popen(
    ["./bin/otter-ppt", "stdio"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    text=True
)

def call(method, params=None, msg_id=1):
    msg = {"jsonrpc": "2.0", "id": msg_id, "method": method}
    if params:
        msg["params"] = params
    proc.stdin.write(json.dumps(msg) + "\n")
    proc.stdin.flush()
    return json.loads(proc.stdout.readline())

# 设置标题
call("tools.call", {"name": "set_title", "arguments": {"title": "My PPT"}}, 1)

# 添加幻灯片
call("tools.call", {"name": "add_slide", "arguments": {"layout": "title"}}, 2)

# 获取状态
result = call("presentation.get", None, 3)
print(json.dumps(result, indent=2, ensure_ascii=False))
```

### Node.js 示例

```javascript
const { spawn } = require('child_process');

const proc = spawn('./bin/otter-ppt', ['stdio']);

let buffer = '';
proc.stdout.on('data', (data) => {
  buffer += data;
  const lines = buffer.split('\n');
  buffer = lines.pop();
  for (const line of lines) {
    if (line.trim()) {
      const response = JSON.parse(line);
      console.log('Response:', response);
    }
  }
});

function send(method, params, id) {
  const msg = { jsonrpc: '2.0', id, method };
  if (params) msg.params = params;
  proc.stdin.write(JSON.stringify(msg) + '\n');
}

send('tools.call', { name: 'set_title', arguments: { title: 'Hello' } }, 1);
send('tools.call', { name: 'add_slide', arguments: { layout: 'title' } }, 2);
send('presentation.get', null, 3);
```
