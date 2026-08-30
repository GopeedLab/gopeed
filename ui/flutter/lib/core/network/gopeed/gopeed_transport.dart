import 'package:dio/dio.dart';

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

abstract interface class GopeedTransport {
  Future<dynamic> request(String path, {String method = 'GET', dynamic data, Map<String, dynamic>? queryParameters});

  Future<Response<String>> proxyRequest(String uri, {dynamic data, Options? options});

  String join(String path);
}

GopeedTransport createGopeedTransport(GopeedTransportConfig config) => createTransport(config);
