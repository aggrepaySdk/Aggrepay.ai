# Aggrepay SDK

Клиенты для [Merchant API](https://aggrepay.ai/docs).  
Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

| | Язык | Установка |
|---|------|-----------|
| [php](php/) | PHP 8.1+ | `composer require aggrepay/sdk` |
| [golang](golang/) | Go 1.21+ | `go get github.com/aggrepaySdk/Aggrepay.ai/golang/aggrepay` |
| [nodejs](nodejs/) | Node.js 18+ | `npm install @aggrepay/sdk` |
| [python](python/) | Python 3.9+ | `pip install aggrepay` |
| [kotlin](kotlin/) | Kotlin / JVM 11+ | см. [kotlin/README.md](kotlin/README.md) |
| [flutter](flutter/aggrepay_sdk/) | Dart / Flutter | см. [flutter/aggrepay_sdk/README.md](flutter/aggrepay_sdk/README.md) |

Документация: https://aggrepay.ai/docs

## Возможности

- HMAC (`X-API-KEY`, `X-TIMESTAMP`, `X-NONCE`, `X-SIGNATURE`)
- Create / get / history инвойса
- `testSecure`
- Проверка callback
- Валидация входа, раздельные ошибки, retry на 429/5xx (`Retry-After`)
- `Idempotency-Key` только на POST, один ключ на вызов (ретраи его не меняют)

## Быстрый старт

1. API-ключ в кабинете мерчанта
2. SDK на своём языке
3. Create → клиент на `pay_url`
4. Callback на `callback_url` → проверка `X-SIGNATURE` → ответ `2xx`

## Важно

- `amount` в минимальных единицах (`10000` = `100.00` RUB)
- `secret_key` только на сервере

## Лицензия

[MIT](LICENSE)
