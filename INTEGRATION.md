# Integration Guide

Otter PPT keeps the core presentation model and tool executor independent from transport code. AI is optional: consumers may configure text/image models in Otter PPT, or run their own models and submit Presentation JSON or granular tool calls. MCP, generic JSON-RPC, REST/OpenAPI, and the Python client all share these protocol boundaries.

## Two AI integration modes

### 1. Otter PPT calls configured models

Configure an OpenAI-compatible text model with `TEXT_MODEL_API_KEY`, `TEXT_MODEL_BASE_URL`, and `TEXT_MODEL_NAME`. Existing `OPENAI_*` variables remain supported as fallbacks. Optionally configure image generation with `IMAGE_MODEL_API_KEY`, `IMAGE_MODEL_BASE_URL`, and `IMAGE_MODEL_NAME`.

Text generation uses the tool-calling agent. When the agent calls `add_image` with `image_prompt`, the configured image model resolves that prompt into a local image asset. If no image model is configured, callers should provide `image_path` themselves.

### 2. External AI supplies the result

Otter PPT can run with no model credentials:

- Submit a complete Presentation object to `POST /api/v1/build` and receive PPTX bytes.
- Submit ordered tool calls to `POST /api/v1/execute`; the response contains updated Presentation JSON that can be inspected, extended, and sent to `/build`.
- Use MCP or STDIO and let Claude Code, Cursor, Codex, or your own orchestrator call the tools directly.

Example external tool batch:

```json
{
  "calls": [
    {"name":"set_title","arguments":{"title":"AI Trends"}},
    {"name":"add_slide","arguments":{"layout":"title"}}
  ]
}
```

Pass the previous `presentation` back with the next batch to continue editing without server-side HTTP session state.

## MCP: Claude Code, Cursor and compatible clients

Build the binary, then register it as a stdio MCP server:

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

The server exposes every granular presentation tool plus:

- `export_pptx`: writes the current session to an editable `.pptx` file.
- `reset_session`: starts a clean presentation session.

A typical agent flow is `set_theme` → `add_slide` → design tools → `export_pptx`. Session state lives for the lifetime of the MCP process, so each client process is isolated.

## Generic STDIO JSON-RPC

Run:

```bash
otter-ppt stdio
```

Messages are newline-delimited JSON-RPC 2.0. Supported methods:

- `tools.list`
- `tools.call` with `{ "name": string, "arguments": object }`
- `presentation.get`
- `session.reset`

Example requests:

```json
{"jsonrpc":"2.0","id":1,"method":"tools.call","params":{"name":"add_slide","arguments":{"layout":"title"}}}
{"jsonrpc":"2.0","id":2,"method":"presentation.get"}
```

This mode is suitable for Codex wrappers, editor extensions, desktop applications, and custom orchestrators that can spawn a subprocess but do not implement MCP.

## REST and OpenAPI

Start the API:

```bash
otter-ppt serve --port 8080
```

The machine-readable contract is [`openapi.yaml`](./openapi.yaml). Generate clients with any OpenAPI 3 generator. Stable entry points are:

- `POST /api/v1/generate`: AI-driven generation.
- `POST /api/v1/execute`: apply externally generated tool calls without an internal model.
- `POST /api/v1/build`: deterministic Presentation JSON → PPTX.
- `GET /api/v1/tools`: portable tool definitions.
- `GET /api/v1/download?id=...`: one-time download returned by `/generate`.
- `GET /health`: readiness check.

Use `/build` when your own software or model produces Presentation JSON; it does not require an LLM API key.

## Local service

Build the Go binary and run `otter-ppt serve --port 8080`. Set `TEXT_MODEL_API_KEY`, `TEXT_MODEL_BASE_URL`, and `TEXT_MODEL_NAME` when using `/generate` (`OPENAI_*` aliases remain supported). Image generation optionally uses `IMAGE_MODEL_API_KEY`, `IMAGE_MODEL_BASE_URL`, and `IMAGE_MODEL_NAME`. The deterministic `/execute` and `/build` endpoints remain usable without model credentials.

## Python

```bash
pip install ./sdk/python
```

```python
from otter_ppt import OtterPPT

client = OtterPPT("http://localhost:8080")
result = client.generate("AI trends", slides=8, language="en")
client.download(result["download_url"], "ai-trends.pptx")
client.close()
```

For deterministic rendering:

```python
client.build(presentation_dict, "output.pptx")
```

## Building a custom integration

Prefer these boundaries:

1. **Presentation JSON** for persistent data and cross-language interchange.
2. **Tool definitions + tool calls** for agentic editing.
3. **PPTX build endpoint** for rendering.
4. **MCP** for editor/CLI agents.

Do not depend on internal Go packages from another application. Use the protocol contracts instead so Otter PPT can evolve without breaking integrations. Never pass API keys inside presentation JSON or tool arguments; configure credentials only through process environment or your own secret manager.
