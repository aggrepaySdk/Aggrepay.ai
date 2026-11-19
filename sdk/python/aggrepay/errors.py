from __future__ import annotations

from typing import Any


class AggrepayError(Exception):
    def __init__(
        self,
        message: str,
        *,
        code: str | None = None,
        http_status: int = 0,
        payload: Any = None,
        retry_after: float = 0.0,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.http_status = http_status
        self.payload = payload
        self.retry_after = retry_after


class ValidationError(AggrepayError):
    """Аргументы до HTTP."""


class TransportError(AggrepayError):
    """Сеть, таймаут, битый JSON, лимит тела."""


class ApiError(AggrepayError):
    """Ответ API."""
