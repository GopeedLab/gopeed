import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

class LocationKeepAlive {
  static const _channel = MethodChannel('gopeed/location_keep_alive');

  static Future<bool> requestPermission() async {
    if (kIsWeb || !Platform.isIOS) return false;

    try {
      final result = await _channel.invokeMethod<bool>('requestPermission');

      return result ?? false;
    } catch (_) {
      return false;
    }
  }

  static Future<void> start() async {
    if (kIsWeb || !Platform.isIOS) return;

    await _channel.invokeMethod('start');
  }

  static Future<void> stop() async {
    if (kIsWeb || !Platform.isIOS) return;

    await _channel.invokeMethod('stop');
  }
}
