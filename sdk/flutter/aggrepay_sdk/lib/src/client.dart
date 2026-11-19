import 'dart:convert';
import 'dart:math';

import 'package:http/http.dart' as http;

import 'errors.dart';
import 'signer.dart' as signer;

const _userAgent = 'aggrepay-dart/1.0.0';
const _maxBody = 2 * 1024 * 1024;
const _currencies = {'RUB', 'USD', 'EUR'};

void _assertBaseUrl(String baseUrl) {
  final u = Uri.parse(baseUrl);
  final host = u.host.toLowerCase();
  final local = host == 'localhost' || host == '127.0.0.1' || host == '::1';
  if (u.scheme != 'https' && !(u.scheme == 'http' && local)) {
    throw ValidationException('only https is allowed (http for localhost)');
  }
}

int _parseRetryAfterMs(String? v) {
  if (v == null || v.isEmpty) return 0;
  final sec = int.tryParse(v);
  if (sec != null) return sec < 0 ? 0 : sec * 1000;
  try {
    final t = DateTime.parse(v).millisecondsSinceEpoch;
    final d = t - DateTime.now().millisecondsSinceEpoch;
    return d < 0 ? 0 : d;
  } catch (_) {
    return 0;
  }
}

class AggrepayClient {
  static const defaultBaseUrl = 'https://aggrepay.ai';

  AggrepayClient(
    this.apiKey,
    this.secretKey, {
    String baseUrl = defaultBaseUrl,
    this.timeout = const Duration(seconds: 30),
    this.maxRetries = 2,
    http.Client? httpClient,
  })  : baseUrl = baseUrl.replaceAll(RegExp(r'/+$'), ''),
        _http = httpClient ?? http.Client() {
    if (apiKey.isEmpty || secretKey.isEmpty) {
      throw ValidationException('apiKey and secretKey are required');
    }
    _assertBaseUrl(this.baseUrl);
  }

  final String apiKey;
  final String secretKey;
  final String baseUrl;
  final Duration timeout;
  final int maxRetries;
  final http.Client _http;

  Future<Map<String, dynamic>> testSecure() async {
    // Ключ один на вызов, ретраи используют тот же Idempotency-Key.
    final key = 'test-secure-${_nonce(8)}';
    final res = await request(
      'POST',
      '/api/v1/merchant/test-secure',
      <String, dynamic>{},
      key,
    );
    return Map<String, dynamic>.from(res['data'] as Map);
  }

  Future<Map<String, dynamic>> createInvoice(
    Map<String, dynamic> params, {
    String? idempotencyKey,
  }) async {
    _validateCreate(params);
    final key = idempotencyKey ?? 'create-${params['order_id']}';
    final res = await request('POST', '/api/v1/invoice/create', params, key);
    return Map<String, dynamic>.from(res['data'] as Map);
  }

  Future<Map<String, dynamic>> getInvoice(String invoiceId) async {
    if (invoiceId.trim().isEmpty) {
      throw ValidationException('invoice_id required');
    }
    final res = await request('GET', '/api/v1/invoice/$invoiceId', null, null);
    return Map<String, dynamic>.from(res['data'] as Map);
  }

  Future<List<dynamic>> invoiceHistory(String invoiceId) async {
    if (invoiceId.trim().isEmpty) {
      throw ValidationException('invoice_id required');
    }
    final res =
        await request('GET', '/api/v1/invoice/$invoiceId/history', null, null);
    return List<dynamic>.from(res['data'] as List);
  }

  bool verifyCallback(
    List<int> rawBody,
    String timestamp,
    String signature,
  ) =>
      signer.verifyCallback(secretKey, timestamp, rawBody, signature);

  Future<Map<String, dynamic>> request(
    String method,
    String path,
    Map<String, dynamic>? jsonBody,
    String? idempotencyKey,
  ) async {
    final m = method.toUpperCase();
    final bodyBytes =
        jsonBody == null ? <int>[] : utf8.encode(jsonEncode(jsonBody));
    // idempotencyKey фиксирован до цикла.
    final attempts = 1 + (maxRetries < 0 ? 0 : maxRetries);
    AggrepayException? last;
    for (var i = 0; i < attempts; i++) {
      if (i > 0) {
        final wait = (last != null && last!.retryAfterMs > 0)
            ? last!.retryAfterMs
            : 200 * (1 << (i - 1));
        await Future<void>.delayed(Duration(milliseconds: wait));
      }
      try {
        return await _roundTrip(m, path, bodyBytes, idempotencyKey);
      } on AggrepayException catch (e) {
        last = e;
        if (!_retryable(e) || i == attempts - 1) rethrow;
      }
    }
    throw last ?? TransportException('request failed');
  }

  Future<Map<String, dynamic>> _roundTrip(
    String method,
    String path,
    List<int> bodyBytes,
    String? idempotencyKey,
  ) async {
    final timestamp =
        (DateTime.now().millisecondsSinceEpoch ~/ 1000).toString();
    final nonce = _nonce(16);
    final canonical =
        signer.requestCanonical(method, path, timestamp, nonce, bodyBytes);
    final signature = signer.sign(secretKey, canonical);

    final headers = <String, String>{
      'Accept': 'application/json',
      'User-Agent': _userAgent,
      'X-API-KEY': apiKey,
      'X-TIMESTAMP': timestamp,
      'X-NONCE': nonce,
      'X-SIGNATURE': signature,
    };
    if (idempotencyKey != null && idempotencyKey.isNotEmpty) {
      headers['Idempotency-Key'] = idempotencyKey;
    }
    if (bodyBytes.isNotEmpty) {
      headers['Content-Type'] = 'application/json';
    }

    late http.Response res;
    try {
      final req = http.Request(method, Uri.parse('$baseUrl$path'))
        ..headers.addAll(headers)
        ..bodyBytes = bodyBytes;
      final streamed = await _http.send(req).timeout(timeout);
      res = await http.Response.fromStream(streamed);
    } catch (e) {
      throw TransportException('HTTP request failed: $e');
    }

    final retryAfterMs = _parseRetryAfterMs(res.headers['retry-after']);
    if (res.bodyBytes.length > _maxBody) {
      throw TransportException(
        'response body too large',
        httpStatus: res.statusCode,
        retryAfterMs: retryAfterMs,
      );
    }

    Map<String, dynamic> decoded;
    try {
      decoded = jsonDecode(res.body) as Map<String, dynamic>;
    } catch (_) {
      throw TransportException(
        'invalid JSON response',
        httpStatus: res.statusCode,
        payload: res.body.length > 256 ? res.body.substring(0, 256) : res.body,
        retryAfterMs: retryAfterMs,
      );
    }

    final apiStatus = decoded['status'];
    if (res.statusCode >= 400 || apiStatus == 'error') {
      final err = decoded['error'];
      final map =
          err is Map ? Map<String, dynamic>.from(err) : Map<String, dynamic>.from(decoded);
      throw ApiException(
        (map['message'] as String?) ?? 'API error',
        code: map['code'] as String?,
        httpStatus: res.statusCode,
        payload: decoded,
        retryAfterMs: retryAfterMs,
      );
    }
    if (apiStatus != null && apiStatus != '' && apiStatus != 'success') {
      throw TransportException(
        'unexpected status $apiStatus',
        httpStatus: res.statusCode,
        payload: decoded,
        retryAfterMs: retryAfterMs,
      );
    }
    if (!decoded.containsKey('data') || decoded['data'] == null) {
      throw TransportException(
        'empty data in success response',
        httpStatus: res.statusCode,
        payload: decoded,
        retryAfterMs: retryAfterMs,
      );
    }
    return decoded;
  }

  static void _validateCreate(Map<String, dynamic> params) {
    final amount = params['amount'];
    if (amount is! int || amount < 1 || amount > 30000000) {
      throw ValidationException('amount must be int between 1 and 30000000');
    }
    if (!_currencies.contains(params['currency'])) {
      throw ValidationException('currency must be RUB, USD or EUR');
    }
    if ((params['order_id']?.toString() ?? '').trim().isEmpty) {
      throw ValidationException('order_id required');
    }
  }

  bool _retryable(Object e) {
    if (e is TransportException && e.httpStatus == 0) return true;
    if (e is AggrepayException) {
      return e.httpStatus == 429 || e.httpStatus >= 500;
    }
    return false;
  }

  static String _nonce(int byteLen) {
    final rnd = Random.secure();
    final bytes = List<int>.generate(byteLen, (_) => rnd.nextInt(256));
    return bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
  }
}
