<?php

declare(strict_types=1);

namespace Aggrepay\Sdk;

final class Signer
{
    public static function bodySha256Hex(string $body): string
    {
        return hash('sha256', $body);
    }

    /** METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256(body) */
    public static function requestCanonical(string $method, string $path, string $timestamp, string $nonce, string $body): string
    {
        return strtoupper($method) . "\n"
            . $path . "\n"
            . $timestamp . "\n"
            . $nonce . "\n"
            . self::bodySha256Hex($body);
    }

    public static function sign(string $secret, string $canonical): string
    {
        return hash_hmac('sha256', $canonical, $secret);
    }

    /** Path всегда /merchant/callback. */
    public static function callbackCanonical(string $timestamp, string $body): string
    {
        return "POST\n/merchant/callback\n"
            . $timestamp . "\n"
            . self::bodySha256Hex($body);
    }

    public static function verifyCallback(string $secret, string $timestamp, string $body, string $signature): bool
    {
        if ($secret === '' || $timestamp === '' || $signature === '') {
            return false;
        }
        $expected = self::sign($secret, self::callbackCanonical($timestamp, $body));
        return hash_equals(strtolower($expected), strtolower($signature));
    }
}
