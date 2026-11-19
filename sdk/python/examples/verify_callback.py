#!/usr/bin/env python3
"""Minimal Flask-less callback check  -  paste into your web framework."""

from __future__ import annotations

import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from aggrepay import AggrepayClient

client = AggrepayClient(
    os.getenv("AGGREPAY_API_KEY", "YOUR_API_KEY"),
    os.getenv("AGGREPAY_SECRET_KEY", "YOUR_SECRET_KEY"),
)


def handle_callback(raw_body: bytes, headers: dict[str, str]) -> tuple[int, dict]:
    ts = headers.get("X-TIMESTAMP") or headers.get("x-timestamp") or ""
    sig = headers.get("X-SIGNATURE") or headers.get("x-signature") or ""
    if not client.verify_callback(raw_body, ts, sig):
        return 401, {"ok": False, "error": "invalid signature"}
    # parse json, update order, then:
    return 200, {"ok": True}


if __name__ == "__main__":
    print("Import handle_callback from this module in your HTTP stack.")
