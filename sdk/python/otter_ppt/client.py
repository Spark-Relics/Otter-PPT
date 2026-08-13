from __future__ import annotations

from pathlib import Path
from typing import Any, Mapping

import httpx


class OtterPPTError(RuntimeError):
    """Raised when the Otter PPT service rejects a request."""


class OtterPPT:
    """Synchronous thin client for the stable Otter PPT HTTP API."""

    def __init__(self, base_url: str = "http://localhost:8080", timeout: float = 120.0) -> None:
        self._client = httpx.Client(base_url=base_url.rstrip("/"), timeout=timeout)


    def close(self) -> None:
        self._client.close()

    def health(self) -> dict[str, Any]:
        return self._json(self._client.get("/health"))

    def tools(self) -> dict[str, Any]:
        return self._json(self._client.get("/api/v1/tools"))

    def generate(
        self,
        topic: str,
        *,
        slides: int = 8,
        language: str = "zh",
        style: str = "",
    ) -> dict[str, Any]:
        response = self._client.post(
            "/api/v1/generate",
            json={"topic": topic, "slides": slides, "language": language, "style": style},
        )
        return self._json(response)

    def execute(
        self,
        calls: list[Mapping[str, Any]],
        *,
        presentation: Mapping[str, Any] | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {"calls": [dict(call) for call in calls]}
        if presentation is not None:
            payload["presentation"] = dict(presentation)
        return self._json(self._client.post("/api/v1/execute", json=payload))

    def build(self, presentation: Mapping[str, Any], output: str | Path) -> Path:
        response = self._client.post("/api/v1/build", json=dict(presentation))
        self._raise_for_status(response)
        path = Path(output)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(response.content)
        return path.resolve()

    def download(self, download_url: str, output: str | Path) -> Path:
        response = self._client.get(download_url)
        self._raise_for_status(response)
        path = Path(output)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(response.content)
        return path.resolve()

    @staticmethod
    def _raise_for_status(response: httpx.Response) -> None:
        try:
            response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            try:
                detail = response.json().get("error", response.text)
            except ValueError:
                detail = response.text
            raise OtterPPTError(str(detail)) from exc

    def _json(self, response: httpx.Response) -> dict[str, Any]:
        self._raise_for_status(response)
        data = response.json()
        if not isinstance(data, dict):
            raise OtterPPTError("expected a JSON object response")
        return data
