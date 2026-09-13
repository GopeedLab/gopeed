import 'package:flutter/services.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';

// Private host/plugin bridge. These settings never originate from extension JS.
class WebViewProfileSettings extends InAppWebViewSettings {
  WebViewProfileSettings({required this.profileId, required bool debug, super.userAgent})
    : super(javaScriptEnabled: true, transparentBackground: true, isInspectable: debug);

  final String profileId;

  @override
  Map<String, dynamic> toMap() => {...super.toMap(), 'gopeedProfileId': profileId};
}

class WebViewProfile {
  WebViewProfile(this.id);

  final String id;
  static const _channel = MethodChannel('com.pichillilorenzo/flutter_inappwebview_cookiemanager');

  Future<void> prepare(String proxyUrl) async {
    await _call('gopeed.prepareProfile', {'proxyUrl': proxyUrl});
  }

  Future<dynamic> _call(String method, Map<String, dynamic> arguments) =>
      _channel.invokeMethod(method, {...arguments, 'gopeedProfileId': id});

  Future<List<Cookie>> getCookies({required WebUri url}) async {
    final values = await _call('getCookies', {'url': url.toString()}) as List? ?? [];
    return values.map((value) => Cookie.fromMap(Map<String, dynamic>.from(value as Map))!).toList();
  }

  Future<void> setCookie({
    required WebUri url,
    required String name,
    required String value,
    String? domain,
    String path = '/',
    int? expiresDate,
    bool isSecure = false,
    bool isHttpOnly = false,
  }) async {
    await _call('setCookie', {
      'url': url.toString(),
      'name': name,
      'value': value,
      'domain': domain,
      'path': path,
      'expiresDate': expiresDate?.toString(),
      'isSecure': isSecure,
      'isHttpOnly': isHttpOnly,
    });
  }

  Future<void> deleteCookie({required WebUri url, required String name, String? domain, String path = '/'}) async {
    await _call('deleteCookie', {'url': url.toString(), 'name': name, 'domain': domain, 'path': path});
  }

  Future<void> deleteAllCookies() async {
    await _call('deleteAllCookies', {});
  }
}
