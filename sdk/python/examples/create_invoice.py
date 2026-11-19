#!/usr/bin/env python3
from __future__ import annotations

import os
import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from aggrepay import AggrepayClient

client = AggrepayClient(
    os.getenv("AGGREPAY_API_KEY", "YOUR_API_KEY"),
    os.getenv("AGGREPAY_SECRET_KEY", "YOUR_SECRET_KEY"),
)

print("testSecure:", client.test_secure())

invoice = client.create_invoice(
    {
        "amount": 10000,
        "currency": "RUB",
        "order_id": f"order-{int(time.time())}",
        "description": "Demo order",
        "project_url": "https://shop.example",
        "redirect_url": "https://shop.example/success",
        "fail_redirect_url": "https://shop.example/fail",
        "callback_url": "https://shop.example/payment/callback",
    }
)
print(invoice)
