import os
from typing import Any

import httpx


GATEWAY_URL = os.getenv("GATEWAY_URL", "http://gateway:8080").rstrip("/")
USER_DETAIL_PATH_TEMPLATE = os.getenv(
    "USER_DETAIL_PATH_TEMPLATE",
    "/api/v1/users/{user_id}",
)


class UserGatewayClient:
    def __init__(self) -> None:
        self._client = httpx.AsyncClient(base_url=GATEWAY_URL, timeout=3.0)

    async def close(self) -> None:
        await self._client.aclose()

    async def fetch_user(self, user_id: str) -> tuple[dict[str, Any] | None, str | None]:
        path = USER_DETAIL_PATH_TEMPLATE.format(user_id=user_id)
        try:
            response = await self._client.get(path)
            response.raise_for_status()
            return response.json(), None
        except httpx.HTTPStatusError as exc:
            return None, f"user {user_id}: gateway returned {exc.response.status_code}"
        except httpx.RequestError as exc:
            return None, f"user {user_id}: gateway request failed ({exc.__class__.__name__})"
        except ValueError:
            return None, f"user {user_id}: gateway returned invalid JSON"
