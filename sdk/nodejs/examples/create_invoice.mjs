import { AggrepayClient } from '../src/index.js';

const client = new AggrepayClient(
  process.env.AGGREPAY_API_KEY || 'YOUR_API_KEY',
  process.env.AGGREPAY_SECRET_KEY || 'YOUR_SECRET_KEY',
);

const ping = await client.testSecure();
console.log('testSecure:', ping);

const invoice = await client.invoices().create({
  amount: 10000,
  currency: 'RUB',
  order_id: `order-${Date.now()}`,
  description: 'Demo order',
  project_url: 'https://shop.example',
  redirect_url: 'https://shop.example/success',
  fail_redirect_url: 'https://shop.example/fail',
  callback_url: 'https://shop.example/payment/callback',
});

console.log(invoice);
