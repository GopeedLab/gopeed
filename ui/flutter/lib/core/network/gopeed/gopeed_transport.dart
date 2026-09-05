import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import 'entry/gopeed_transport_stub.dart'
    if (dart.library.html) 'entry/gopeed_transport_web.dart'
    if (dart.library.io) 'entry/gopeed_transport_native.dart';

class GopeedTransportConfig {
  const GopeedTransportConfig({
    required this.network,
    required this.address,
    required this.apiToken,
    this.onUnauthorized,
  });

  final String network;
  final String address;
  final String apiToken;
  final void Function()? onUnauthorized;
}

const defaultWebApiPort = 9999;

/// Resolves the API base URL used by Flutter Web.
///
/// This preserves the legacy behavior: debug builds connect to the local
/// backend on port 9999, while release builds use the page's server origin.
String webApiBaseUrl(Uri pageUri, {bool? debugMode}) {
  final useDebugPort = debugMode ?? kDebugMode;
  if (useDebugPort) return 'http://127.0.0.1:$defaultWebApiPort/';
  return '${pageUri.origin}/';
}

/// Returns the Dio base URL, preserving relative requests in release builds.
String webTransportBaseUrl(String address, Uri pageUri, {bool? debugMode}) {
  final useDebugPort = debugMode ?? kDebugMode;
  return useDebugPort ? normalizeWebApiBaseUrl(address, pageUri) : '';
}

/// Normalizes either an absolute API URL or a host:port address.
String normalizeWebApiBaseUrl(String address, Uri pageUri) {
  var value = address.trim();
  if (value.isEmpty) return '${pageUri.origin}/';

  final parsed = Uri.tryParse(value);
  if (parsed != null && parsed.hasScheme && parsed.host.isNotEmpty) {
    value = Uri(
      scheme: parsed.scheme,
      userInfo: parsed.userInfo,
      host: parsed.host,
      port: parsed.hasPort ? parsed.port : null,
      path: parsed.path,
    ).toString();
  } else {
    value = '${pageUri.scheme}://${value.replaceAll(RegExp(r'/+$'), '')}';
  }
  return value.endsWith('/') ? value : '$value/';
}

abstract interface class GopeedTransport {
  Future<dynamic> request(String path, {String method = 'GET', dynamic data, Map<String, dynamic>? queryParameters});

  Future<Response<String>> proxyRequest(String uri, {dynamic data, Options? options});

  String join(String path);
}

GopeedTransport createGopeedTransport(GopeedTransportConfig config) => createTransport(config);
