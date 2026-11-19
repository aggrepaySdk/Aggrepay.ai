# Aggrepay Flutter / Dart SDK

Клиент [Merchant API](https://aggrepay.ai/docs).

`secret_key` только на сервере. Не кладите его в мобильное приложение.

Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

## Установка

```yaml
dependencies:
  aggrepay_sdk:
    git:
      url: https://github.com/aggrepaySdk/Aggrepay.ai.git
      path: flutter/aggrepay_sdk
```

## Использование

```dart
import 'package:aggrepay_sdk/aggrepay_sdk.dart';

final client = AggrepayClient('YOUR_API_KEY', 'YOUR_SECRET_KEY');

await client.testSecure();

final invoice = await client.createInvoice({
  'amount': 10000,
  'currency': 'RUB',
  'order_id': 'order-1001',
  'description': 'Оплата заказа #1001',
  'project_url': 'https://shop.example',
  'redirect_url': 'https://shop.example/success',
  'callback_url': 'https://shop.example/payment/callback',
});

final got = await client.getInvoice(invoice['invoice_id'] as String);
final hist = await client.invoiceHistory(invoice['invoice_id'] as String);
```

### Callback

```dart
if (!client.verifyCallback(rawBody, timestamp, signature)) {
  // 401
}
```

### Ошибки

`ValidationException`, `TransportException`, `ApiException`.
`Idempotency-Key` только на POST, один ключ на вызов.
Повторы на 429/5xx с учётом `Retry-After`.

## Пример

```bash
export AGGREPAY_API_KEY=...
export AGGREPAY_SECRET_KEY=...
dart pub get
dart run example/create_invoice.dart
```

## Лицензия

MIT
