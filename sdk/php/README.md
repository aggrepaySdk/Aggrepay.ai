# Aggrepay PHP SDK

Клиент [Merchant API](https://aggrepay.ai/docs). PHP 8.1+.

Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

## Установка

```bash
composer require aggrepay/sdk
```

## Использование

```php
use Aggrepay\Sdk\AggrepayClient;

$client = new AggrepayClient('YOUR_API_KEY', 'YOUR_SECRET_KEY');

$ok = $client->testSecure();

$invoice = $client->invoices()->create([
    'amount' => 10000, // 100.00 RUB
    'currency' => 'RUB',
    'order_id' => 'order-1001',
    'description' => 'Оплата заказа #1001',
    'project_url' => 'https://shop.example',
    'redirect_url' => 'https://shop.example/success',
    'callback_url' => 'https://shop.example/payment/callback',
]);

$got = $client->invoices()->get($invoice['invoice_id']);
$history = $client->invoices()->history($invoice['invoice_id']);
```

### Callback

```php
$raw = file_get_contents('php://input');
$ts = $_SERVER['HTTP_X_TIMESTAMP'] ?? '';
$sig = $_SERVER['HTTP_X_SIGNATURE'] ?? '';

if (!$client->verifyCallback($raw, $ts, $sig)) {
    http_response_code(401);
    exit;
}
```

Каноника:

```text
POST
/merchant/callback
{X-TIMESTAMP}
{SHA256_HEX(raw_body)}
```

### Ошибки

- `ValidationException` - до HTTP
- `TransportException` - сеть, таймаут, битый JSON
- `ApiException` - ответ API

`Idempotency-Key` только на POST, один ключ на вызов.
Повторы на 429/5xx с учётом `Retry-After`.

## Примеры

```bash
export AGGREPAY_API_KEY=...
export AGGREPAY_SECRET_KEY=...
composer install
php examples/create_invoice.php
```

## Лицензия

MIT
