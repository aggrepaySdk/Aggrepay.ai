from __future__ import annotations

import json
import secrets
import time
import urllib.error
import urllib.parse
import urllib.request
from email.utils import parsedate_to_datetime
from typing import Any

from .errors import AggrepayError, ApiError, TransportError, ValidationError
from .signer import request_canonical, sign, verify_callback

_CURRENCIES = frozenset({"RUB", "USD", "EUR"})
_USER_AGENT = "aggrepay-python/1.0.0"
_MAX_BODY = 2 * 1024 * 1024


def _assert_base_url(base_url: str) -> None:
    p = urllib.parse.urlparse(base_url)
    if not p.scheme or not p.netloc:
        raise ValidationError("invalid base_url")
    host = (p.hostname or "").lower()
    local = host in {"localhost", "127.0.0.1", "::1"}
    if p.scheme != "https" and not (p.scheme == "http" and local):
        raise ValidationError("only https is allowed (http for localhost)")


def _parse_retry_after(v: str | None) -> float:
    if not v:
        return 0.0
    try:
        return max(0.0, float(v))
    except ValueError:
        pass
    try:
        dt = parsedate_to_datetime(v)
        return max(0.0, dt.timestamp() - time.time())
    except Exception:  # noqa: BLE001
        return 0.0


class AggrepayClient:
    DEFAULT_BASE_URL = "https://aggrepay.ai"

    def __init__(
        self,
        api_key: str,
        secret_key: str,
        *,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = 30.0,
        max_retries: int = 2,
    ) -> None:
        if not api_key or not secret_key:
            raise ValidationError("api_key and secret_key are required")
        self.api_key = api_key
        self.secret_key = secret_key
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.max_retries = max_retries
        _assert_base_url(self.base_url)

    def test_secure(self) -> dict[str, Any]:
        # Ключ один на вызов, ретраи используют тот же Idempotency-Key.
        key = f"test-secure-{secrets.token_hex(8)}"
        res = self.request("POST", "/api/v1/merchant/test-secure", {}, key)
        return res["data"]

    def create_invoice(self, params: dict[str, Any], idempotency_key: str | None = None) -> dict[str, Any]:
        self._validate_create(params)
        key = idempotency_key or f"create-{params['order_id']}"
        res = self.request("POST", "/api/v1/invoice/create", params, key)
        return res["data"]

    def get_invoice(self, invoice_id: str) -> dict[str, Any]:
        if not str(invoice_id).strip():
            raise ValidationError("invoice_id required")
        res = self.request("GET", f"/api/v1/invoice/{invoice_id}", None, None)
        return res["data"]

    def invoice_history(self, invoice_id: str) -> list[dict[str, Any]]:
        if not str(invoice_id).strip():
            raise ValidationError("invoice_id required")
        res = self.request("GET", f"/api/v1/invoice/{invoice_id}/history", None, None)
        return res["data"]

    def verify_callback(self, raw_body: bytes | str, timestamp: str, signature: str) -> bool:
        return verify_callback(self.secret_key, timestamp, raw_body, signature)

    def request(
        self,
        method: str,
        path: str,
        json_body: dict[str, Any] | None,
        idempotency_key: str | None,
    ) -> dict[str, Any]:
        method = method.upper()
        if json_body is None:
            body = b""
        else:
            body = json.dumps(json_body, ensure_ascii=False, separators=(",", ":")).encode("utf-8")

        # idempotency_key фиксирован до цикла.
        attempts = 1 + max(0, self.max_retries)
        last: AggrepayError | None = None
        for i in range(attempts):
            if i > 0 and last is not None:
                wait = last.retry_after if last.retry_after > 0 else 0.2 * (2 ** (i - 1))
                time.sleep(wait)
            try:
                return self._round_trip(method, path, body, idempotency_key)
            except (TransportError, ApiError) as e:
                last = e
                if not self._retryable(e) or i == attempts - 1:
                    raise
        assert last is not None
        raise last

    def _round_trip(
        self,
        method: str,
        path: str,
        body: bytes,
        idempotency_key: str | None,
    ) -> dict[str, Any]:
        timestamp = str(int(time.time()))
        nonce = secrets.token_hex(16)
        canonical = request_canonical(method, path, timestamp, nonce, body)
        signature = sign(self.secret_key, canonical)

        headers = {
            "Accept": "application/json",
            "User-Agent": _USER_AGENT,
            "X-API-KEY": self.api_key,
            "X-TIMESTAMP": timestamp,
            "X-NONCE": nonce,
            "X-SIGNATURE": signature,
        }
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        if body:
            headers["Content-Type"] = "application/json"

        req = urllib.request.Request(
            self.base_url + path,
            data=body if body else None,
            headers=headers,
            method=method,
        )
        retry_after = 0.0
        try:
            with urllib.request.urlopen(req, timeout=self.timeout) as resp:
                retry_after = _parse_retry_after(resp.headers.get("Retry-After"))
                raw = resp.read(_MAX_BODY + 1)
                status = resp.status
        except urllib.error.HTTPError as e:
            retry_after = _parse_retry_after(e.headers.get("Retry-After") if e.headers else None)
            raw = e.read(_MAX_BODY + 1)
            status = e.code
        except Exception as e:  # noqa: BLE001
            raise TransportError(f"HTTP request failed: {e}") from e

        if len(raw) > _MAX_BODY:
            raise TransportError("response body too large", http_status=status, retry_after=retry_after)

        try:
            decoded = json.loads(raw.decode("utf-8"))
        except Exception as exc:  # noqa: BLE001
            raise TransportError(
                "invalid JSON response",
                http_status=status,
                payload=raw[:256],
                retry_after=retry_after,
            ) from exc

        api_status = decoded.get("status")
        if status >= 400 or api_status == "error":
            err = decoded.get("error") if isinstance(decoded.get("error"), dict) else decoded
            raise ApiError(
                err.get("message") if isinstance(err, dict) else "API error",
                code=err.get("code") if isinstance(err, dict) else None,
                http_status=status,
                payload=decoded,
                retry_after=retry_after,
            )
        if api_status not in (None, "", "success"):
            raise TransportError(
                f"unexpected status {api_status}",
                http_status=status,
                payload=decoded,
                retry_after=retry_after,
            )
        if "data" not in decoded or decoded["data"] is None:
            raise TransportError(
                "empty data in success response",
                http_status=status,
                payload=decoded,
                retry_after=retry_after,
            )
        return decoded

    @staticmethod
    def _validate_create(params: dict[str, Any]) -> None:
        amount = params.get("amount")
        if not isinstance(amount, int) or isinstance(amount, bool) or amount < 1 or amount > 30_000_000:
            raise ValidationError("amount must be int between 1 and 30000000")
        if params.get("currency") not in _CURRENCIES:
            raise ValidationError("currency must be RUB, USD or EUR")
        if not str(params.get("order_id", "")).strip():
            raise ValidationError("order_id required")

    @staticmethod
    def _retryable(e: AggrepayError) -> bool:
        if isinstance(e, TransportError) and e.http_status == 0:
            return True
        return e.http_status == 429 or e.http_status >= 500
