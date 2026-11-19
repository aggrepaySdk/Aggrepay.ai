<?php

declare(strict_types=1);

namespace Aggrepay\Sdk\Service;

use Aggrepay\Sdk\AggrepayClient;
use Aggrepay\Sdk\ValidationException;

final class InvoiceService
{
    private const CURRENCIES = ['RUB', 'USD', 'EUR'];

    public function __construct(private readonly AggrepayClient $client)
    {
    }

    /**
     * @param array{
     *   amount:int,
     *   currency:string,
     *   order_id:string,
     *   description?:string,
     *   project_url?:string,
     *   redirect_url?:string,
     *   fail_redirect_url?:string,
     *   callback_url?:string
     * } $params
     * @return array{invoice_id:string,pay_url:string,payment_status:string}
     */
    public function create(array $params, ?string $idempotencyKey = null): array
    {
        $this->validateCreate($params);
        $key = $idempotencyKey ?? ('create-' . $params['order_id']);
        $res = $this->client->request('POST', '/api/v1/invoice/create', $params, $key);
        return $res['data'];
    }

    /**
     * @return array{
     *   invoice_id:string,
     *   merchant_id:string,
     *   order_id:string,
     *   amount:int,
     *   currency:string,
     *   status:string,
     *   expires_at:string,
     *   created_at:string,
     *   updated_at:string
     * }
     */
    public function get(string $invoiceId): array
    {
        $invoiceId = trim($invoiceId);
        if ($invoiceId === '') {
            throw new ValidationException('invoice_id required');
        }
        $res = $this->client->request('GET', '/api/v1/invoice/' . rawurlencode($invoiceId), null, null);
        return $res['data'];
    }

    /** @return list<array{old_status:?string,new_status:string,source:string,created_at:string,reason?:string}> */
    public function history(string $invoiceId): array
    {
        $invoiceId = trim($invoiceId);
        if ($invoiceId === '') {
            throw new ValidationException('invoice_id required');
        }
        $res = $this->client->request(
            'GET',
            '/api/v1/invoice/' . rawurlencode($invoiceId) . '/history',
            null,
            null,
        );
        return $res['data'];
    }

    /** @param array<string,mixed> $params */
    private function validateCreate(array $params): void
    {
        $amount = $params['amount'] ?? null;
        if (!is_int($amount) || $amount < 1 || $amount > 30_000_000) {
            throw new ValidationException('amount must be int between 1 and 30000000');
        }
        $currency = (string) ($params['currency'] ?? '');
        if (!in_array($currency, self::CURRENCIES, true)) {
            throw new ValidationException('currency must be RUB, USD or EUR');
        }
        $orderId = trim((string) ($params['order_id'] ?? ''));
        if ($orderId === '') {
            throw new ValidationException('order_id required');
        }
    }
}
