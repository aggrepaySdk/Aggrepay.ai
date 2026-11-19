# Aggrepay Go SDK

Клиент [Merchant API](https://aggrepay.ai/docs). Go 1.21+.

Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

## Установка

```bash
go get github.com/aggrepaySdk/Aggrepay.ai/golang/aggrepay
```

## Использование

```go
import (
    "context"
    "github.com/aggrepaySdk/Aggrepay.ai/golang/aggrepay"
)

client, err := aggrepay.NewClient(apiKey, secretKey)
// опции: aggrepay.WithBaseURL(...), WithHTTPClient(...), WithMaxRetries(2)

ping, err := client.TestSecure(ctx)

inv, err := client.CreateInvoice(ctx, aggrepay.CreateInvoiceParams{
    Amount:      10000, // 100.00 RUB
    Currency:    aggrepay.CurrencyRUB,
    OrderID:     "order-1001",
    Description: "Оплата заказа #1001",
    ProjectURL:  "https://shop.example",
    RedirectURL: "https://shop.example/success",
    CallbackURL: "https://shop.example/payment/callback",
}, "")

got, err := client.GetInvoice(ctx, inv.InvoiceID)
hist, err := client.InvoiceHistory(ctx, inv.InvoiceID)
```

### Callback

```go
ok := client.VerifyCallback(rawBody, r.Header.Get("X-TIMESTAMP"), r.Header.Get("X-SIGNATURE"))
```

Каноника:

```text
POST
/merchant/callback
{X-TIMESTAMP}
{SHA256_HEX(raw_body)}
```

### Ошибки

- `*ValidationError` - до HTTP
- `*TransportError` - сеть, таймаут, битый JSON, слишком большое тело
- `*APIError` - ответ API (`Code`, `HTTPStatus`, `RetryAfter`)

`Idempotency-Key` только на POST, один ключ на вызов (ретраи не меняют его).
Повторы: 429/5xx; если есть `Retry-After`, ждём его.

## Пример

```bash
export AGGREPAY_API_KEY=...
export AGGREPAY_SECRET_KEY=...
go run ./examples/create_invoice.go
```

## Лицензия

MIT
