class AggrepayException implements Exception {
  AggrepayException(
    this.message, {
    this.code,
    this.httpStatus = 0,
    this.payload,
    this.retryAfterMs = 0,
  });

  final String message;
  final String? code;
  final int httpStatus;
  final Object? payload;
  final int retryAfterMs;

  @override
  String toString() =>
      code == null ? 'AggrepayException: $message' : 'AggrepayException: $message ($code)';
}

/** Аргументы до HTTP. */
class ValidationException extends AggrepayException {
  ValidationException(super.message);
}

/** Сеть, таймаут, битый JSON, лимит тела. */
class TransportException extends AggrepayException {
  TransportException(
    super.message, {
    super.httpStatus,
    super.payload,
    super.retryAfterMs,
  });
}

/** Ответ API. */
class ApiException extends AggrepayException {
  ApiException(
    super.message, {
    super.code,
    super.httpStatus,
    super.payload,
    super.retryAfterMs,
  });
}
