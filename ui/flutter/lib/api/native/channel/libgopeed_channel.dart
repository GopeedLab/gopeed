import 'package:flutter/services.dart';

import '../libgopeed_interface.dart';

class LibgopeedChannel implements LibgopeedAbi {
  static const _channel = MethodChannel('gopeed.com/libgopeed');

  @override
  Future<void> init(String cfg) async {
    await _channel.invokeMethod<String>('init', {
      'cfg': cfg,
    });
  }

  @override
  Future<String> invoke(String params) async {
    final result = await _channel.invokeMethod<String>('invoke', {
      'params': params,
    });
    return result!;
  }
}
