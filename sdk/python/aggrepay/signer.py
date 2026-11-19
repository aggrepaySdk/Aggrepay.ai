from __future__ import annotations

import hashlib
import hmac


def body_sha256_hex(body: bytes | str) -> str:
    if isinstance(body, str):
        body = body.encode("utf-8")
    return hashlib.sha256(body).hexdigest()


def request_canonical(
    method: str,
    path: str,
    timestamp: str,
    nonce: str,
    body: bytes | str,
) -> str:
    return (
        f"{method.upper()}\n"
        f"{path}\n"
        f"{timestamp}\n"
        f"{nonce}\n"
        f"{body_sha256_hex(body)}"
    )


def sign(secret: str, canonical: str) -> str:
    return hmac.new(secret.encode("utf-8"), canonical.encode("utf-8"), hashlib.sha256).hexdigest()


def callback_canonical(timestamp: str, body: bytes | str) -> str:
    return f"POST\n/merchant/callback\n{timestamp}\n{body_sha256_hex(body)}"


def verify_callback(secret: str, timestamp: str, body: bytes | str, signature: str) -> bool:
    if not secret or not timestamp or not signature:
        return False
    expected = sign(secret, callback_canonical(timestamp, body))
    return hmac.compare_digest(expected.lower(), str(signature).lower())
