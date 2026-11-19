import 'dart:io';

import 'package:aggrepay_sdk/aggrepay_sdk.dart';

Future<void> main() async {
  final apiKey = Platform.environment['AGGREPAY_API_KEY'];
  final secretKey = Platform.environment['AGGREPAY_SECRET_KEY'];
  if (apiKey == null || secretKey == null) {
    stderr.writeln('AGGREPAY_API_KEY and AGGREPAY_SECRET_KEY are required');
    exit(1);
  }

  final client = AggrepayClient(apiKey, secretKey);

  final secure = await client.testSecure();
  print('testSecure: $secure');

  final invoice = await client.createInvoice({
    'amount': 10000,
    'currency': 'RUB',
    'order_id': 'order-flutter-${DateTime.now().millisecondsSinceEpoch}',
    'description': 'Flutter SDK example',
    'callback_url': 'https://shop.example/payment/callback',
  });

  print('pay_url: ${invoice['pay_url']}');
  print('invoice_id: ${invoice['invoice_id']}');
}
