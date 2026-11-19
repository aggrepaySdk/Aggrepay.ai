<?php

declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

use Aggrepay\Sdk\AggrepayClient;

$apiKey = getenv('AGGREPAY_API_KEY') ?: 'YOUR_API_KEY';
$secret = getenv('AGGREPAY_SECRET_KEY') ?: 'YOUR_SECRET_KEY';

$client = new AggrepayClient($apiKey, $secret);

// 1) Check HMAC wiring
$ping = $client->testSecure();
echo "testSecure: {$ping['message']} (merchant {$ping['merchant_id']})\n";

// 2) Create invoice (amount in minor units: 10000 = 100.00 RUB)
$invoice = $client->invoices()->create([
    'amount' => 10000,
    'currency' => 'RUB',
    'order_id' => 'order-' . time(),
    'description' => 'Demo order',
    'project_url' => 'https://shop.example',
    'redirect_url' => 'https://shop.example/success',
    'fail_redirect_url' => 'https://shop.example/fail',
    'callback_url' => 'https://shop.example/payment/callback',
]);

echo "invoice_id: {$invoice['invoice_id']}\n";
echo "pay_url:    {$invoice['pay_url']}\n";
echo "status:     {$invoice['payment_status']}\n";

// Redirect the customer to pay_url
