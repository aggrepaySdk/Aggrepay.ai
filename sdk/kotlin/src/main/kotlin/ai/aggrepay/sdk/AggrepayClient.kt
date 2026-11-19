package ai.aggrepay.sdk

import org.json.JSONArray
import org.json.JSONObject
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.security.SecureRandom
import java.time.Duration
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.time.format.DateTimeParseException

class AggrepayClient(
    private val apiKey: String,
    private val secretKey: String,
    baseUrl: String = DEFAULT_BASE_URL,
    timeoutSeconds: Long = 30,
    private val maxRetries: Int = 2,
) {
    private val baseUrl: String = baseUrl.trimEnd('/')
    private val timeout: Duration = Duration.ofSeconds(timeoutSeconds)
    private val http: HttpClient = HttpClient.newBuilder()
        .connectTimeout(timeout)
        .build()

    init {
        if (apiKey.isBlank() || secretKey.isBlank()) {
            throw ValidationException("apiKey and secretKey are required")
        }
        assertBaseUrl(this.baseUrl)
    }

    fun testSecure(): JSONObject {
        // Ключ один на вызов, ретраи используют тот же Idempotency-Key.
        val key = "test-secure-${nonce(8)}"
        val res = request("POST", "/api/v1/merchant/test-secure", JSONObject(), key)
        return res.getJSONObject("data")
    }

    fun createInvoice(params: Map<String, Any?>, idempotencyKey: String? = null): JSONObject {
        validateCreate(params)
        val orderId = params["order_id"].toString()
        val key = idempotencyKey ?: "create-$orderId"
        val body = JSONObject(params.filterValues { it != null })
        val res = request("POST", "/api/v1/invoice/create", body, key)
        return res.getJSONObject("data")
    }

    fun getInvoice(invoiceId: String): JSONObject {
        if (invoiceId.isBlank()) throw ValidationException("invoice_id required")
        val res = request("GET", "/api/v1/invoice/$invoiceId", null, null)
        return res.getJSONObject("data")
    }

    fun invoiceHistory(invoiceId: String): JSONArray {
        if (invoiceId.isBlank()) throw ValidationException("invoice_id required")
        val res = request("GET", "/api/v1/invoice/$invoiceId/history", null, null)
        return res.getJSONArray("data")
    }

    fun verifyCallback(rawBody: ByteArray, timestamp: String, signature: String): Boolean =
        Signer.verifyCallback(secretKey, timestamp, rawBody, signature)

    fun verifyCallback(rawBody: String, timestamp: String, signature: String): Boolean =
        verifyCallback(rawBody.toByteArray(Charsets.UTF_8), timestamp, signature)

    fun request(
        method: String,
        path: String,
        jsonBody: JSONObject?,
        idempotencyKey: String?,
    ): JSONObject {
        val m = method.uppercase()
        val bodyBytes = if (jsonBody == null) ByteArray(0) else jsonBody.toString().toByteArray(Charsets.UTF_8)
        // idempotencyKey фиксирован до цикла.
        val attempts = 1 + maxRetries.coerceAtLeast(0)
        var last: AggrepayException? = null
        for (i in 0 until attempts) {
            if (i > 0) {
                val wait = last?.retryAfterMs?.takeIf { it > 0 } ?: (200L * (1L shl (i - 1)))
                Thread.sleep(wait)
            }
            try {
                return roundTrip(m, path, bodyBytes, idempotencyKey)
            } catch (e: AggrepayException) {
                last = e
                if (!retryable(e) || i == attempts - 1) throw e
            }
        }
        throw last ?: TransportException("request failed")
    }

    private fun roundTrip(
        method: String,
        path: String,
        bodyBytes: ByteArray,
        idempotencyKey: String?,
    ): JSONObject {
        val timestamp = (System.currentTimeMillis() / 1000).toString()
        val nonce = nonce(16)
        val canonical = Signer.requestCanonical(method, path, timestamp, nonce, bodyBytes)
        val signature = Signer.sign(secretKey, canonical)

        val builder = HttpRequest.newBuilder()
            .uri(URI.create("$baseUrl$path"))
            .timeout(timeout)
            .header("Accept", "application/json")
            .header("User-Agent", USER_AGENT)
            .header("X-API-KEY", apiKey)
            .header("X-TIMESTAMP", timestamp)
            .header("X-NONCE", nonce)
            .header("X-SIGNATURE", signature)

        if (!idempotencyKey.isNullOrBlank()) {
            builder.header("Idempotency-Key", idempotencyKey)
        }
        if (bodyBytes.isNotEmpty()) {
            builder.header("Content-Type", "application/json")
            builder.method(method, HttpRequest.BodyPublishers.ofByteArray(bodyBytes))
        } else {
            builder.method(method, HttpRequest.BodyPublishers.noBody())
        }

        val response = try {
            http.send(builder.build(), HttpResponse.BodyHandlers.ofByteArray())
        } catch (e: Exception) {
            throw TransportException("HTTP request failed: ${e.message}", cause = e)
        }

        val retryAfterMs = parseRetryAfterMs(response.headers().firstValue("Retry-After").orElse(null))
        val rawBytes = response.body()
        val status = response.statusCode()
        if (rawBytes.size > MAX_BODY) {
            throw TransportException("response body too large", httpStatus = status, retryAfterMs = retryAfterMs)
        }
        val raw = String(rawBytes, Charsets.UTF_8)

        val decoded = try {
            JSONObject(raw)
        } catch (e: Exception) {
            throw TransportException(
                "invalid JSON response",
                httpStatus = status,
                payload = raw.take(256),
                retryAfterMs = retryAfterMs,
                cause = e,
            )
        }

        val apiStatus = decoded.optString("status")
        if (status >= 400 || apiStatus == "error") {
            val err = decoded.optJSONObject("error")
            throw ApiException(
                err?.optString("message")?.ifBlank { null }
                    ?: decoded.optString("message").ifBlank { "API error" },
                code = err?.optString("code")?.ifBlank { null }
                    ?: decoded.optString("code").ifBlank { null },
                httpStatus = status,
                payload = decoded,
                retryAfterMs = retryAfterMs,
            )
        }
        if (apiStatus.isNotEmpty() && apiStatus != "success") {
            throw TransportException(
                "unexpected status $apiStatus",
                httpStatus = status,
                payload = decoded,
                retryAfterMs = retryAfterMs,
            )
        }
        if (!decoded.has("data") || decoded.isNull("data")) {
            throw TransportException(
                "empty data in success response",
                httpStatus = status,
                payload = decoded,
                retryAfterMs = retryAfterMs,
            )
        }
        return decoded
    }

    private fun retryable(e: AggrepayException): Boolean {
        if (e is TransportException && e.httpStatus == 0) return true
        return e.httpStatus == 429 || e.httpStatus >= 500
    }

    private fun validateCreate(params: Map<String, Any?>) {
        val amount = params["amount"]
        val amountLong = when (amount) {
            is Number -> amount.toLong()
            else -> -1L
        }
        if (amountLong < 1 || amountLong > 30_000_000L) {
            throw ValidationException("amount must be between 1 and 30000000")
        }
        val currency = params["currency"]?.toString()
        if (currency !in CURRENCIES) {
            throw ValidationException("currency must be RUB, USD or EUR")
        }
        if (params["order_id"]?.toString()?.trim().isNullOrEmpty()) {
            throw ValidationException("order_id required")
        }
    }

    companion object {
        const val DEFAULT_BASE_URL = "https://aggrepay.ai"
        private const val USER_AGENT = "aggrepay-kotlin/1.0.0"
        private const val MAX_BODY = 2 * 1024 * 1024
        private val CURRENCIES = setOf("RUB", "USD", "EUR")
        private val rng = SecureRandom()

        private fun nonce(byteLen: Int): String {
            val bytes = ByteArray(byteLen)
            rng.nextBytes(bytes)
            return bytes.joinToString("") { "%02x".format(it) }
        }

        private fun assertBaseUrl(baseUrl: String) {
            val uri = try {
                URI(baseUrl)
            } catch (e: Exception) {
                throw ValidationException("invalid baseUrl")
            }
            val host = (uri.host ?: "").lowercase()
            val local = host == "localhost" || host == "127.0.0.1" || host == "::1"
            val scheme = (uri.scheme ?: "").lowercase()
            if (scheme != "https" && !(scheme == "http" && local)) {
                throw ValidationException("only https is allowed (http for localhost)")
            }
        }

        private fun parseRetryAfterMs(v: String?): Long {
            if (v.isNullOrBlank()) return 0
            v.toLongOrNull()?.let { return maxOf(0L, it * 1000) }
            return try {
                val t = ZonedDateTime.parse(v, DateTimeFormatter.RFC_1123_DATE_TIME)
                maxOf(0L, t.toInstant().toEpochMilli() - System.currentTimeMillis())
            } catch (_: DateTimeParseException) {
                0
            }
        }
    }
}
