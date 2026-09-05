import 'dart:async';
import 'dart:io';

import 'package:flutter/services.dart';

import '../../api/api.dart' as api;
import '../../api/model/task.dart';

class LocationKeepAlive {
  const LocationKeepAlive._();

  static const _channel = MethodChannel('gopeed/location_keep_alive');

  static Future<bool> requestPermission() async {
    if (!Platform.isIOS) return false;
    try {
      return await _channel.invokeMethod<bool>('requestPermission') ?? false;
    } catch (_) {
      return false;
    }
  }

  static Future<void> start() async {
    if (Platform.isIOS) await _channel.invokeMethod<void>('start');
  }

  static Future<void> stop() async {
    if (Platform.isIOS) await _channel.invokeMethod<void>('stop');
  }
}

class LocationKeepAliveCoordinator {
  LocationKeepAliveCoordinator._();

  static final instance = LocationKeepAliveCoordinator._();

  Timer? _timer;
  bool Function()? _enabled;
  bool _reconciling = false;

  void start(bool Function() enabled) {
    if (!Platform.isIOS) return;
    _enabled = enabled;
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 10), (_) => unawaited(reconcile()));
    unawaited(reconcile());
  }

  Future<void> reconcile({bool? enabled}) async {
    if (!Platform.isIOS || _reconciling) return;
    _reconciling = true;
    try {
      if (!(enabled ?? _enabled?.call() ?? false)) {
        await LocationKeepAlive.stop();
        return;
      }
      final running = await api.getTasks([Status.running]);
      running.isEmpty ? await LocationKeepAlive.stop() : await LocationKeepAlive.start();
    } catch (_) {
      // Keep the current native state when task status cannot be queried.
    } finally {
      _reconciling = false;
    }
  }
}
