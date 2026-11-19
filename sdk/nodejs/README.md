# Aggrepay Node.js SDK

Клиент [Merchant API](https://aggrepay.ai/docs). Node.js 18+.

Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

## Установка

```bash
npm install @aggrepay/sdk
```

## Использование

```js
import { AggrepayClient } from '@aggrepay/sdk';

const client = new AggrepayClient(process.env.AGGREPAY_API_KEY, process.env.AGGREPAY_SECRET_KEY);

await client.testSecure();

const invoice = await client.invoices().create({
  amount: 10000, // 100.00 RUB
  currency: 'RUB',
  order_id: 'order-1001',
  description: 'Оплата заказа #1001',
  project_url: 'https://shop.example',
  redirect_url: 'https://shop.example/success',
  callback_url: 'https://shop.example/payment/callback',
});

const got = await client.invoices().get(invoice.invoice_id);
const hist = await client.invoices().history(invoice.invoice_id);
```

### Callback

```js
const ok = client.verifyCallback(rawBody, req.headers['x-timestamp'], req.headers['x-signature']);
if (!ok) return res.status(401).end();
```

### Ошибки

`ValidationError`, `TransportError`, `ApiError`.
`Idempotency-Key` только на POST, один ключ на вызов.
Повторы на 429/5xx с учётом `Retry-After`.

## Примеры

```bash
export AGGREPAY_API_KEY=...
export AGGREPAY_SECRET_KEY=...
node examples/create_invoice.mjs
```

## Лицензия

MIT
