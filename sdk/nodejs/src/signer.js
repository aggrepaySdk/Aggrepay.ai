import { createHash, createHmac, timingSafeEqual } from 'node:crypto';

export function bodySha256Hex(body) {
  return createHash('sha256').update(body ?? '', 'utf8').digest('hex');
}

/** METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256(body) */
export function requestCanonical(method, path, timestamp, nonce, body) {
  return [
    String(method).toUpperCase(),
    path,
    String(timestamp),
    String(nonce),
    bodySha256Hex(body ?? ''),
  ].join('\n');
}

export function sign(secret, canonical) {
  return createHmac('sha256', secret).update(canonical, 'utf8').digest('hex');
}

/** Path всегда /merchant/callback. */
export function callbackCanonical(timestamp, body) {
  return ['POST', '/merchant/callback', String(timestamp), bodySha256Hex(body ?? '')].join('\n');
}

export function verifyCallback(secret, timestamp, body, signature) {
  if (!secret || !timestamp || !signature) return false;
  const expected = sign(secret, callbackCanonical(timestamp, body));
  const a = Buffer.from(expected.toLowerCase(), 'utf8');
  const b = Buffer.from(String(signature).toLowerCase(), 'utf8');
  if (a.length !== b.length) return false;
  return timingSafeEqual(a, b);
}
