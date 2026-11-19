package ai.aggrepay.sdk

open class AggrepayException(
    message: String,
    val code: String? = null,
    val httpStatus: Int = 0,
    val payload: Any? = null,
    val retryAfterMs: Long = 0,
) : RuntimeException(message)

/** Аргументы до HTTP. */
class ValidationException(message: String) : AggrepayException(message)

/** Сеть, таймаут, битый JSON, лимит тела. */
class TransportException(
    message: String,
    httpStatus: Int = 0,
    payload: Any? = null,
    retryAfterMs: Long = 0,
    cause: Throwable? = null,
) : AggrepayException(message, httpStatus = httpStatus, payload = payload, retryAfterMs = retryAfterMs) {
    init {
        if (cause != null) initCause(cause)
    }
}

/** Ответ API. */
class ApiException(
    message: String,
    code: String? = null,
    httpStatus: Int = 0,
    payload: Any? = null,
    retryAfterMs: Long = 0,
) : AggrepayException(message, code, httpStatus, payload, retryAfterMs)
