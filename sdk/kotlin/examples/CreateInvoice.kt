package examples

import ai.aggrepay.sdk.AggrepayClient

fun main() {
    val apiKey = System.getenv("AGGREPAY_API_KEY")
        ?: error("AGGREPAY_API_KEY is required")
    val secretKey = System.getenv("AGGREPAY_SECRET_KEY")
        ?: error("AGGREPAY_SECRET_KEY is required")

    val client = AggrepayClient(apiKey, secretKey)

    val secure = client.testSecure()
    println("testSecure: $secure")

    val invoice = client.createInvoice(
        mapOf(
            "amount" to 10000,
            "currency" to "RUB",
            "order_id" to "order-kotlin-${System.currentTimeMillis()}",
            "description" to "Kotlin SDK example",
            "callback_url" to "https://shop.example/payment/callback",
        ),
    )
    println("pay_url: ${invoice.getString("pay_url")}")
    println("invoice_id: ${invoice.getString("invoice_id")}")
}
