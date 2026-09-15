import 'dart:io';

import 'package:flutter/services.dart';

class ContinuedProcessing {
  const ContinuedProcessing._();

  static const _channel =
      MethodChannel('gopeed/continued_processing');

  static Future<bool> isSupported() async {
    if (!Platform.isIOS) return false;

    try {
      return await _channel.invokeMethod<bool>(
            'isSupported',
          ) ??
          false;
    } catch (_) {
      return false;
    }
  }

  static Future<bool> setEnabled(
    bool enabled,
  ) async {
    if (!Platform.isIOS) return false;

    try {
      return await _channel.invokeMethod<bool>(
            'setEnabled',
            {'enabled': enabled},
          ) ??
          false;
    } catch (_) {
      return false;
    }
  }
}
