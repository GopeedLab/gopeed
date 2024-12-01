import 'package:flutter/services.dart';

import '../libgopeed_interface.dart';

class LibgopeedChannel implements LibgopeedAbi {
  static const _channel = MethodChannel('gopeed.com/libgopeed');

  @override
  Future<String> create(String cfg) async {
    final result = await _channel.invokeMethod<String>('create', {
      'cfg': cfg,
    });
    return result!;
  }

  @override
  Future<String> invoke(int instance, String params) async {
    final result = await _channel.invokeMethod<String>('invoke', {
      'instance': instance,
      'params': params,
    });
    return result!;
  }
}
