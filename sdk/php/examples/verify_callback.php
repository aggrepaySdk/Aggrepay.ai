<?php

declare(strict_types=1);

/**
 * Minimal callback endpoint example.
 * Put behind HTTPS. Always verify signature before trusting status.
 */

require __DIR__ . '/../vendor/autoload.php';

use Aggrepay\Sdk\AggrepayClient;

$apiKey = getenv('AGGREPAY_API_KEY') ?: 'YOUR_API_KEY';
$secret = getenv('AGGREPAY_SECRET_KEY') ?: 'YOUR_SECRET_KEY';

$client = new AggrepayClient($apiKey, $secret);

$rawBody = file_get_contents('php://input') ?: '';
$timestamp = $_SERVER['HTTP_X_TIMESTAMP'] ?? '';
$signature = $_SERVER['HTTP_X_SIGNATURE'] ?? '';

if ($timestamp === '' || $signature === '' || !$client->verifyCallback($rawBody, $timestamp, $signature)) {
    http_response_code(401);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'invalid signature']);
    exit;
}

$payload = json_decode($rawBody, true);
// обновите заказ по $payload['order_id'] / $payload['status']

http_response_code(200);
header('Content-Type: application/json');
echo json_encode(['ok' => true]);
