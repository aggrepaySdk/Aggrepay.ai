# Aggrepay Kotlin SDK

Клиент [Merchant API](https://aggrepay.ai/docs). Kotlin, JVM 11+.

Репозиторий: [github.com/aggrepaySdk/Aggrepay.ai](https://github.com/aggrepaySdk/Aggrepay.ai)

## Установка

```kotlin
// settings.gradle.kts
include(":aggrepay-sdk")
project(":aggrepay-sdk").projectDir = file("../sdk/kotlin")

// build.gradle.kts
dependencies {
    implementation(project(":aggrepay-sdk"))
}
```

Нужен `org.json:json`.

## Использование

```kotlin
import ai.aggrepay.sdk.AggrepayClient

val client = AggrepayClient(apiKey = "YOUR_API_KEY", secretKey = "YOUR_SECRET_KEY")

client.testSecure()

val invoice = client.createInvoice(
    mapOf(
        "amount" to 10000,
        "currency" to "RUB",
        "order_id" to "order-1001",
        "description" to "Оплата заказа #1001",
        "project_url" to "https://shop.example",
        "redirect_url" to "https://shop.example/success",
        "callback_url" to "https://shop.example/payment/callback",
    ),
)

val got = client.getInvoice(invoice.getString("invoice_id"))
val hist = client.invoiceHistory(invoice.getString("invoice_id"))
```

### Callback

```kotlin
if (!client.verifyCallback(rawBody, timestamp, signature)) {
    // 401
}
```

### Ошибки

`ValidationException`, `TransportException`, `ApiException`.
`Idempotency-Key` только на POST, один ключ на вызов.
Повторы на 429/5xx с учётом `Retry-After`.

## Пример

```bash
export AGGREPAY_API_KEY=...
export AGGREPAY_SECRET_KEY=...
./gradlew build
```

## Лицензия

MIT
