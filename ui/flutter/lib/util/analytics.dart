import 'dart:io';

import 'package:device_info_plus/device_info_plus.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'log_util.dart';
import 'package_info.dart';

/// GA4 Measurement Protocol configuration supplied by the release build.
class AnalyticsConfig {
  const AnalyticsConfig._();

  static const measurementId = String.fromEnvironment('GA4_MEASUREMENT_ID');
  static const apiSecret = String.fromEnvironment('GA4_API_SECRET');

  static bool get isConfigured => measurementId.isNotEmpty && apiSecret.isNotEmpty;
}

/// Sends only anonymous application events. Its preference and fallback
/// client ID are persisted by the Go downloader configuration.
class Analytics {
  Analytics._();

  static final instance = Analytics._();

  late final String _clientId;
  late final int _sessionId;
  late final Dio _dio;
  bool _initialized = false;

  String get clientId => _initialized ? _clientId : '';

  Future<void> init({String fallbackClientId = ''}) async {
    if (_initialized || !AnalyticsConfig.isConfigured) return;
    _clientId = await _getDeviceId(fallbackClientId);
    _sessionId = DateTime.now().millisecondsSinceEpoch ~/ 1000;
    _dio = Dio(BaseOptions(connectTimeout: const Duration(seconds: 10), receiveTimeout: const Duration(seconds: 10)));
    _initialized = true;
  }

  Future<void> logAppOpen() => logEvent('app_open');

  Future<void> logEvent(String name, [Map<String, dynamic>? params]) async {
    if (!_initialized) return;
    final payload = {
      'client_id': _clientId,
      'timestamp_micros': DateTime.now().microsecondsSinceEpoch,
      'events': [
        {
          'name': name,
          'params': {
            'session_id': _sessionId.toString(),
            'engagement_time_msec': 100,
            'platform': _platformName(),
            'app_version': appVersion,
            ...?params,
          },
        },
      ],
    };
    try {
      await _dio.post(
        'https://www.google-analytics.com/mp/collect',
        queryParameters: {'measurement_id': AnalyticsConfig.measurementId, 'api_secret': AnalyticsConfig.apiSecret},
        data: payload,
      );
    } catch (error) {
      logger.w('GA4Analytics: failed to send event "$name": $error');
    }
  }

  Future<String> _getDeviceId(String fallbackClientId) async {
    String? deviceId;
    try {
      final info = DeviceInfoPlugin();
      if (kIsWeb) {
        deviceId = null;
      } else if (Platform.isAndroid) {
        deviceId = (await info.androidInfo).id;
      } else if (Platform.isIOS) {
        deviceId = (await info.iosInfo).identifierForVendor;
      } else if (Platform.isMacOS) {
        deviceId = (await info.macOsInfo).systemGUID;
      } else if (Platform.isWindows) {
        deviceId = (await info.windowsInfo).deviceId;
      } else if (Platform.isLinux) {
        deviceId = (await info.linuxInfo).machineId;
      }
    } catch (error) {
      logger.w('GA4Analytics: failed to read device id: $error');
    }

    if (deviceId == null || deviceId.isEmpty) {
      deviceId = fallbackClientId;
      if (deviceId.isEmpty) {
        final random = DateTime.now().microsecondsSinceEpoch % 2147483647;
        final timestamp = DateTime.now().millisecondsSinceEpoch ~/ 1000;
        deviceId = '$random.$timestamp';
      }
    }
    return deviceId;
  }

  String _platformName() {
    if (kIsWeb) return 'web';
    if (Platform.isAndroid) return 'android';
    if (Platform.isIOS) return 'ios';
    if (Platform.isMacOS) return 'macos';
    if (Platform.isWindows) return 'windows';
    if (Platform.isLinux) return 'linux';
    return 'unknown';
  }
}
