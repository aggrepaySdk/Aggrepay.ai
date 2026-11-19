import { randomBytes } from 'node:crypto';
import { ApiError, TransportError, ValidationError } from './errors.js';
import { requestCanonical, sign, verifyCallback } from './signer.js';

const USER_AGENT = 'aggrepay-node/1.0.0';
const MAX_BODY = 2 * 1024 * 1024;
const CURRENCIES = new Set(['RUB', 'USD', 'EUR']);

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

function validateCreate(params) {
  const amount = params?.amount;
  if (!Number.isInteger(amount) || amount < 1 || amount > 30_000_000) {
    throw new ValidationError('amount must be int between 1 and 30000000');
  }
  if (!CURRENCIES.has(params.currency)) {
    throw new ValidationError('currency must be RUB, USD or EUR');
  }
  if (!String(params.order_id || '').trim()) {
    throw new ValidationError('order_id required');
  }
}

function assertBaseUrl(baseUrl) {
  let u;
  try {
    u = new URL(baseUrl);
  } catch {
    throw new ValidationError('invalid baseUrl');
  }
  const host = u.hostname;
  const local = host === 'localhost' || host === '127.0.0.1' || host === '::1';
  if (u.protocol !== 'https:' && !(u.protocol === 'http:' && local)) {
    throw new ValidationError('only https is allowed (http for localhost)');
  }
}

function parseRetryAfterMs(v) {
  if (!v) return 0;
  const sec = Number(v);
  if (Number.isFinite(sec)) return Math.max(0, sec * 1000);
  const t = Date.parse(v);
  if (Number.isNaN(t)) return 0;
  return Math.max(0, t - Date.now());
}

export class AggrepayClient {
  static DEFAULT_BASE_URL = 'https://aggrepay.ai';

  /**
   * @param {string} apiKey
   * @param {string} secretKey
   * @param {{ baseUrl?: string, timeoutMs?: number, maxRetries?: number }} [options]
   */
  constructor(apiKey, secretKey, options = {}) {
    if (!apiKey || !secretKey) throw new ValidationError('apiKey and secretKey are required');
    this.apiKey = apiKey;
    this.secretKey = secretKey;
    this.baseUrl = (options.baseUrl || AggrepayClient.DEFAULT_BASE_URL).replace(/\/$/, '');
    this.timeoutMs = options.timeoutMs ?? 30_000;
    this.maxRetries = options.maxRetries ?? 2;
    assertBaseUrl(this.baseUrl);
  }

  invoices() {
    const client = this;
    return {
      async create(params, idempotencyKey) {
        validateCreate(params);
        const res = await client.request(
          'POST',
          '/api/v1/invoice/create',
          params,
          idempotencyKey || `create-${params.order_id}`,
        );
        return res.data;
      },
      async get(invoiceId) {
        if (!String(invoiceId || '').trim()) throw new ValidationError('invoice_id required');
        const res = await client.request(
          'GET',
          `/api/v1/invoice/${encodeURIComponent(invoiceId)}`,
          null,
          null,
        );
        return res.data;
      },
      async history(invoiceId) {
        if (!String(invoiceId || '').trim()) throw new ValidationError('invoice_id required');
        const res = await client.request(
          'GET',
          `/api/v1/invoice/${encodeURIComponent(invoiceId)}/history`,
          null,
          null,
        );
        return res.data;
      },
    };
  }

  async testSecure() {
    // Ключ один на вызов, ретраи используют тот же Idempotency-Key.
    const key = `test-secure-${randomBytes(8).toString('hex')}`;
    const res = await this.request('POST', '/api/v1/merchant/test-secure', {}, key);
    return res.data;
  }

  verifyCallback(rawBody, timestamp, signature) {
    return verifyCallback(this.secretKey, timestamp, rawBody, signature);
  }

  async request(method, path, jsonBody, idempotencyKey) {
    const m = String(method).toUpperCase();
    const body = jsonBody == null ? '' : JSON.stringify(jsonBody);
    // idempotencyKey фиксирован до цикла.
    const attempts = 1 + Math.max(0, this.maxRetries);
    let last;
    for (let i = 0; i < attempts; i++) {
      if (i > 0) {
        const wait = last?.retryAfterMs > 0 ? last.retryAfterMs : 200 * 2 ** (i - 1);
        await sleep(wait);
      }
      try {
        return await this.roundTrip(m, path, body, idempotencyKey);
      } catch (e) {
        last = e;
        if (!this.retryable(e) || i === attempts - 1) throw e;
      }
    }
    throw last;
  }

  async roundTrip(method, path, body, idempotencyKey) {
    const timestamp = String(Math.floor(Date.now() / 1000));
    const nonce = randomBytes(16).toString('hex');
    const canonical = requestCanonical(method, path, timestamp, nonce, body);
    const signature = sign(this.secretKey, canonical);

    const headers = {
      Accept: 'application/json',
      'User-Agent': USER_AGENT,
      'X-API-KEY': this.apiKey,
      'X-TIMESTAMP': timestamp,
      'X-NONCE': nonce,
      'X-SIGNATURE': signature,
    };
    if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
    if (body !== '') headers['Content-Type'] = 'application/json';

    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), this.timeoutMs);
    let res;
    try {
      res = await fetch(`${this.baseUrl}${path}`, {
        method,
        headers,
        body: body === '' ? undefined : body,
        signal: ctrl.signal,
      });
    } catch (e) {
      throw new TransportError(`HTTP request failed: ${e.message || e}`, { payload: e });
    } finally {
      clearTimeout(timer);
    }

    const retryAfterMs = parseRetryAfterMs(res.headers.get('retry-after'));
    const raw = await res.text();
    if (raw.length > MAX_BODY) {
      throw new TransportError('response body too large', { httpStatus: res.status, retryAfterMs });
    }

    let decoded;
    try {
      decoded = JSON.parse(raw);
    } catch {
      throw new TransportError('invalid JSON response', {
        httpStatus: res.status,
        payload: raw.slice(0, 256),
        retryAfterMs,
      });
    }

    const apiStatus = decoded.status;
    if (!res.ok || apiStatus === 'error') {
      const err = decoded.error && typeof decoded.error === 'object' ? decoded.error : decoded;
      throw new ApiError(err.message || 'API error', {
        code: err.code,
        httpStatus: res.status,
        payload: decoded,
        retryAfterMs,
      });
    }
    if (apiStatus && apiStatus !== 'success') {
      throw new TransportError(`unexpected status ${apiStatus}`, {
        httpStatus: res.status,
        payload: decoded,
        retryAfterMs,
      });
    }
    if (decoded.data == null) {
      throw new TransportError('empty data in success response', {
        httpStatus: res.status,
        payload: decoded,
        retryAfterMs,
      });
    }
    return decoded;
  }

  retryable(e) {
    if (e instanceof TransportError && !e.httpStatus) return true;
    const s = e.httpStatus || 0;
    return s === 429 || s >= 500;
  }
}
