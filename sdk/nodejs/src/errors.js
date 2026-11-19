export class AggrepayError extends Error {
  constructor(message, { code = null, httpStatus = 0, payload = null, retryAfterMs = 0 } = {}) {
    super(message);
    this.name = 'AggrepayError';
    this.code = code;
    this.httpStatus = httpStatus;
    this.payload = payload;
    this.retryAfterMs = retryAfterMs;
  }
}

/** Аргументы до HTTP. */
export class ValidationError extends AggrepayError {
  constructor(message) {
    super(message);
    this.name = 'ValidationError';
  }
}

/** Сеть, таймаут, битый JSON, лимит тела. */
export class TransportError extends AggrepayError {
  constructor(message, opts = {}) {
    super(message, opts);
    this.name = 'TransportError';
  }
}

/** Ответ API. */
export class ApiError extends AggrepayError {
  constructor(message, opts = {}) {
    super(message, opts);
    this.name = 'ApiError';
  }
}
