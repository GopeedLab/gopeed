import 'package:flutter/services.dart';

///Platform native utilities
class PlatformUtil {
  static PlatformUtil? _instance;
  static const MethodChannel _channel =
      MethodChannel('com.pichillilorenzo/flutter_inappwebview_platformutil');

  PlatformUtil._();

  ///Get [PlatformUtil] instance.
  static PlatformUtil instance() {
    return (_instance != null) ? _instance! : _init();
  }

  static PlatformUtil _init() {
    _channel.setMethodCallHandler((call) async {
      try {
        return await _handleMethod(call);
      } on Error catch (e) {
        print(e);
        print(e.stackTrace);
      }
    });
    _instance = PlatformUtil._();
    return _instance!;
  }

  static Future<dynamic> _handleMethod(MethodCall call) async {}

  String? _cachedSystemVersion;

  ///Get current platform system version.
  Future<String> getSystemVersion() async {
    if (_cachedSystemVersion != null) {
      return _cachedSystemVersion!;
    }
    Map<String, dynamic> args = <String, dynamic>{};
    _cachedSystemVersion =
        await _channel.invokeMethod('getSystemVersion', args);
    return _cachedSystemVersion!;
  }

  ///Format date.
  Future<String> formatDate(
      {required DateTime date,
      required String format,
      String locale = "en_US",
      String timezone = "UTC"}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('date', () => date.millisecondsSinceEpoch);
    args.putIfAbsent('format', () => format);
    args.putIfAbsent('locale', () => locale);
    args.putIfAbsent('timezone', () => timezone);
    return await _channel.invokeMethod('formatDate', args);
  }

  ///Get cookie expiration date used by Web platform.
  Future<String> getWebCookieExpirationDate({required DateTime date}) async {
    Map<String, dynamic> args = <String, dynamic>{};
    args.putIfAbsent('date', () => date.millisecondsSinceEpoch);
    return await _channel.invokeMethod('getWebCookieExpirationDate', args);
  }
}
