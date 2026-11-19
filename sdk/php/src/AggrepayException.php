<?php

declare(strict_types=1);

namespace Aggrepay\Sdk;

use RuntimeException;

class AggrepayException extends RuntimeException
{
    public function __construct(
        string $message,
        public readonly ?string $codeName = null,
        public readonly int $httpStatus = 0,
        public readonly mixed $payload = null,
        public readonly float $retryAfterSec = 0.0,
    ) {
        parent::__construct($message, $httpStatus);
    }
}

/** Аргументы до HTTP. */
final class ValidationException extends AggrepayException
{
}

/** Сеть, таймаут, битый JSON, лимит тела. */
final class TransportException extends AggrepayException
{
}

/** Ответ API. */
final class ApiException extends AggrepayException
{
}
