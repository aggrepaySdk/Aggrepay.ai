<?php

declare(strict_types=1);

namespace Aggrepay\Sdk;

use Aggrepay\Sdk\Service\InvoiceService;

final class AggrepayClient
{
    public const DEFAULT_BASE_URL = 'https://aggrepay.ai';

    private const USER_AGENT = 'aggrepay-php/1.0.0';
    private const MAX_BODY = 2_097_152; // 2 MiB

    public function __construct(
        private readonly string $apiKey,
        private readonly string $secretKey,
        private readonly string $baseUrl = self::DEFAULT_BASE_URL,
        private readonly int $timeoutSec = 30,
        private readonly int $maxRetries = 2,
    ) {
        if (trim($this->apiKey) === '' || trim($this->secretKey) === '') {
            throw new ValidationException('apiKey and secretKey are required');
        }
        $this->assertBaseUrl($this->baseUrl);
    }

    public function invoices(): InvoiceService
    {
        return new InvoiceService($this);
    }

    /** @return array{merchant_id:string,message:string} */
    public function testSecure(): array
    {
        // Ключ один на вызов, ретраи используют тот же Idempotency-Key.
        $key = 'test-secure-' . bin2hex(random_bytes(8));
        $res = $this->request('POST', '/api/v1/merchant/test-secure', new \stdClass(), $key);
        return $res['data'];
    }

    public function verifyCallback(string $rawBody, string $timestamp, string $signature): bool
    {
        return Signer::verifyCallback($this->secretKey, $timestamp, $rawBody, $signature);
    }

    /**
     * @param object|array<string,mixed>|null $jsonBody null = пустое тело (GET)
     * @return array{status:string,data:mixed}
     */
    public function request(string $method, string $path, object|array|null $jsonBody, ?string $idempotencyKey): array
    {
        $method = strtoupper($method);
        if ($jsonBody === null) {
            $body = '';
        } else {
            $encoded = json_encode($jsonBody, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
            if ($encoded === false) {
                throw new TransportException('failed to encode JSON body');
            }
            $body = $encoded;
        }

        // idempotencyKey фиксирован до цикла.
        $attempts = 1 + max(0, $this->maxRetries);
        $last = null;
        for ($i = 0; $i < $attempts; $i++) {
            if ($i > 0) {
                $wait = $this->retryWaitSec($last);
                usleep((int) round($wait * 1_000_000));
            }
            try {
                return $this->roundTrip($method, $path, $body, $idempotencyKey);
            } catch (TransportException|ApiException $e) {
                $last = $e;
                if (!$this->retryable($e) || $i === $attempts - 1) {
                    throw $e;
                }
            }
        }
        throw $last ?? new TransportException('request failed');
    }

    /** @return array{status:string,data:mixed} */
    private function roundTrip(string $method, string $path, string $body, ?string $idempotencyKey): array
    {
        $timestamp = (string) time();
        try {
            $nonce = bin2hex(random_bytes(16));
        } catch (\Throwable $e) {
            throw new TransportException('nonce generation failed: ' . $e->getMessage());
        }

        $canonical = Signer::requestCanonical($method, $path, $timestamp, $nonce, $body);
        $signature = Signer::sign($this->secretKey, $canonical);

        $headers = [
            'Accept: application/json',
            'User-Agent: ' . self::USER_AGENT,
            'X-API-KEY: ' . $this->apiKey,
            'X-TIMESTAMP: ' . $timestamp,
            'X-NONCE: ' . $nonce,
            'X-SIGNATURE: ' . $signature,
        ];
        if ($idempotencyKey !== null && $idempotencyKey !== '') {
            $headers[] = 'Idempotency-Key: ' . $idempotencyKey;
        }
        if ($body !== '') {
            $headers[] = 'Content-Type: application/json';
        }

        $ch = curl_init(rtrim($this->baseUrl, '/') . $path);
        if ($ch === false) {
            throw new TransportException('curl_init failed');
        }

        $headerBag = [];
        $opts = [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_TIMEOUT => $this->timeoutSec,
            CURLOPT_USERAGENT => self::USER_AGENT,
            CURLOPT_HEADERFUNCTION => static function ($ch, string $line) use (&$headerBag): int {
                $len = strlen($line);
                $parts = explode(':', $line, 2);
                if (count($parts) === 2) {
                    $headerBag[strtolower(trim($parts[0]))] = trim($parts[1]);
                }
                return $len;
            },
        ];
        if ($body !== '') {
            $opts[CURLOPT_POSTFIELDS] = $body;
        }
        curl_setopt_array($ch, $opts);

        $raw = curl_exec($ch);
        $errno = curl_errno($ch);
        $err = curl_error($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        $retryAfter = $this->parseRetryAfter($headerBag['retry-after'] ?? null);

        if ($raw === false) {
            throw new TransportException('HTTP request failed: ' . $err, null, 0, null, $retryAfter);
        }
        if (strlen($raw) > self::MAX_BODY) {
            throw new TransportException('response body too large', null, $status, null, $retryAfter);
        }

        /** @var array<string,mixed>|null $decoded */
        $decoded = json_decode($raw, true);
        if (!is_array($decoded)) {
            throw new TransportException('invalid JSON response', null, $status, substr($raw, 0, 256), $retryAfter);
        }

        $apiStatus = (string) ($decoded['status'] ?? '');
        if ($status >= 400 || $apiStatus === 'error') {
            $errObj = is_array($decoded['error'] ?? null) ? $decoded['error'] : $decoded;
            $code = (string) ($errObj['code'] ?? '');
            $msg = (string) ($errObj['message'] ?? 'API error');
            throw new ApiException($msg !== '' ? $msg : 'API error', $code !== '' ? $code : null, $status, $decoded, $retryAfter);
        }
        if ($apiStatus !== '' && $apiStatus !== 'success') {
            throw new TransportException('unexpected status ' . $apiStatus, null, $status, $decoded, $retryAfter);
        }
        if (!array_key_exists('data', $decoded) || $decoded['data'] === null) {
            throw new TransportException('empty data in success response', null, $status, $decoded, $retryAfter);
        }

        return $decoded;
    }

    private function retryable(AggrepayException $e): bool
    {
        if ($e instanceof TransportException && $e->httpStatus === 0) {
            return true;
        }
        $s = $e->httpStatus;
        return $s === 429 || $s >= 500;
    }

    private function retryWaitSec(?AggrepayException $e): float
    {
        if ($e !== null && $e->retryAfterSec > 0) {
            return $e->retryAfterSec;
        }
        return 0.2;
    }

    private function parseRetryAfter(mixed $v): float
    {
        if ($v === null || $v === '') {
            return 0.0;
        }
        if (is_numeric($v)) {
            return max(0.0, (float) $v);
        }
        $ts = strtotime((string) $v);
        if ($ts === false) {
            return 0.0;
        }
        return max(0.0, (float) ($ts - time()));
    }

    private function assertBaseUrl(string $baseUrl): void
    {
        $p = parse_url($baseUrl);
        if ($p === false || empty($p['scheme']) || empty($p['host'])) {
            throw new ValidationException('invalid baseUrl');
        }
        $host = strtolower((string) $p['host']);
        $local = in_array($host, ['localhost', '127.0.0.1', '::1'], true);
        $scheme = strtolower((string) $p['scheme']);
        if ($scheme !== 'https' && !($scheme === 'http' && $local)) {
            throw new ValidationException('only https is allowed (http for localhost)');
        }
    }
}
