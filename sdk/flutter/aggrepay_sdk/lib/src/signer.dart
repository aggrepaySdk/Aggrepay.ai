import 'dart:convert';

import 'package:crypto/crypto.dart';

String bodySha256Hex(List<int> body) => sha256.convert(body).toString();

String bodySha256HexString(String body) => bodySha256Hex(utf8.encode(body));

/// METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256(body)
String requestCanonical(
  String method,
  String path,
  String timestamp,
  String nonce,
  List<int> body,
) {
  return '${method.toUpperCase()}\n'
      '$path\n'
      '$timestamp\n'
      '$nonce\n'
      '${bodySha256Hex(body)}';
}

String sign(String secret, String canonical) {
  final hmac = Hmac(sha256, utf8.encode(secret));
  return hmac.convert(utf8.encode(canonical)).toString();
}

/// Path всегда /merchant/callback.
String callbackCanonical(String timestamp, List<int> body) {
  return 'POST\n/merchant/callback\n$timestamp\n${bodySha256Hex(body)}';
}

bool verifyCallback(
  String secret,
  String timestamp,
  List<int> body,
  String signature,
) {
  if (secret.isEmpty || timestamp.isEmpty || signature.isEmpty) return false;
  final expected = sign(secret, callbackCanonical(timestamp, body));
  return expected.toLowerCase() == signature.toLowerCase();
}
