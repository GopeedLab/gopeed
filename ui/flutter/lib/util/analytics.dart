import 'dart:io';

import 'package:device_info_plus/device_info_plus.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'package:gopeed/database/database.dart';
import 'package:gopeed/util/package_info.dart';

import 'log_util.dart';

/// GA4 Measurement Protocol configuration from dart-define
class Config {
  static const String measurementId =
      String.fromEnvironment('GA4_MEASUREMENT_ID');
  static const String apiSecret = String.fromEnvironment('GA4_API_SECRET');

  static bool get isConfigured =>
      measurementId.isNotEmpty && apiSecret.isNotEmpty;
}

/// GA4 Measurement Protocol Analytics
class Analytics {
  static final Analytics _instance = Analytics._internal();
  static Analytics get instance => _instance;

  Analytics._internal();

  late String _clientId;
  late int _sessionId;
  late Dio _dio;
  bool _initialized = false;

  Future<void> init() async {
    if (_initialized) return;
    if (!Config.isConfigured) return;

    _clientId = await _getDeviceId();
    _dio = Dio(BaseOptions(
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
    ));
    _sessionId = DateTime.now().millisecondsSinceEpoch ~/ 1000;
    _initialized = true;
  }

  Future<String> _getDeviceId() async {
    final deviceInfo = DeviceInfoPlugin();
    String? deviceId;
    try {
      if (kIsWeb) {
        deviceId = null;
      } else if (Platform.isAndroid) {
        final androidInfo = await deviceInfo.androidInfo;
        deviceId = androidInfo.id;
      } else if (Platform.isIOS) {
        final iosInfo = await deviceInfo.iosInfo;
        deviceId = iosInfo.identifierForVendor;
      } else if (Platform.isMacOS) {
        final macInfo = await deviceInfo.macOsInfo;
        deviceId = macInfo.systemGUID;
      } else if (Platform.isWindows) {
        final windowsInfo = await deviceInfo.windowsInfo;
        deviceId = windowsInfo.deviceId;
      } else if (Platform.isLinux) {
        final linuxInfo = await deviceInfo.linuxInfo;
        deviceId = linuxInfo.machineId;
      }
    } catch (e) {
      debugPrint('GA4Analytics: Failed to get device id: $e');
    }

    // Fallback to persisted client id
    if (deviceId == null || deviceId.isEmpty) {
      deviceId = Database.instance.getAnalyticsClientId();
      if (deviceId == null || deviceId.isEmpty) {
        final random = DateTime.now().microsecondsSinceEpoch % 2147483647;
        final timestamp = DateTime.now().millisecondsSinceEpoch ~/ 1000;
        deviceId = '$random.$timestamp';
        Database.instance.saveAnalyticsClientId(deviceId);
      }
    }
    return deviceId;
  }

  String _getPlatform() {
    if (kIsWeb) return 'web';
    if (Platform.isAndroid) return 'android';
    if (Platform.isIOS) return 'ios';
    if (Platform.isMacOS) return 'macos';
    if (Platform.isWindows) return 'windows';
    if (Platform.isLinux) return 'linux';
    return 'unknown';
  }

  Future<void> logAppOpen() async {
    await logEvent('app_open');
  }

  Future<void> logEvent(String name, [Map<String, dynamic>? params]) async {
    if (!_initialized) {
      return;
    }

    final eventParams = <String, dynamic>{
      'session_id': _sessionId.toString(),
      'engagement_time_msec': 100,
      'platform': _getPlatform(),
      'app_version': packageInfo.version,
      ...?params,
    };

    final payload = {
      'client_id': _clientId,
      'timestamp_micros': DateTime.now().microsecondsSinceEpoch,
      'events': [
        {
          'name': name,
          'params': eventParams,
        }
      ],
    };

    try {
      await _dio.post(
        'https://www.google-analytics.com/mp/collect',
        queryParameters: {
          'measurement_id': Config.measurementId,
          'api_secret': Config.apiSecret,
        },
        data: payload,
      );
    } catch (e) {
      logger.w('GA4Analytics: Failed to send event "$name": $e');
    }
  }
}
