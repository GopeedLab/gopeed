import 'dart:convert';

import 'package:flutter/services.dart';

import 'libgopeed_interface.dart';
import 'start_config.dart';

class LibgopeedChannel implements LibgopeedInterface {
  static const _channel = MethodChannel('gopeed.com/libgopeed');

  @override
  Future<int> start(StartConfig cfg) async {
    final port = await _channel.invokeMethod('start', {
      'cfg': jsonEncode(cfg),
    });
    return port as int;
  }

  @override
  Future<void> stop() async {
    return await _channel.invokeMethod('stop');
  }
}
