# Aggrepay Python SDK

Клиент [Merchant API](https://aggrepay.ai/docs). Python 3.9+.

Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

## Установка

```bash
pip install aggrepay
```

## Использование

```python
from aggrepay import AggrepayClient

client = AggrepayClient(api_key, secret_key)

client.test_secure()

invoice = client.create_invoice({
    "amount": 10000,  # 100.00 RUB
    "currency": "RUB",
    "order_id": "order-1001",
    "description": "Оплата заказа #1001",
    "project_url": "https://shop.example",
    "redirect_url": "https://shop.example/success",
    "callback_url": "https://shop.example/payment/callback",
})

got = client.get_invoice(invoice["invoice_id"])
hist = client.invoice_history(invoice["invoice_id"])
```

### Callback

```python
ok = client.verify_callback(raw_body, request.headers["X-TIMESTAMP"], request.headers["X-SIGNATURE"])
```

### Ошибки

`ValidationError`, `TransportError`, `ApiError`.
`Idempotency-Key` только на POST, один ключ на вызов.
Повторы на 429/5xx с учётом `Retry-After`.

## Примеры

```bash
export AGGREPAY_API_KEY=...
export AGGREPAY_SECRET_KEY=...
pip install -e .
python examples/create_invoice.py
```

## Лицензия

MIT
