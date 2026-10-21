import 'package:dio/dio.dart';

import '../gopeed_transport.dart';

GopeedTransport createTransport(GopeedTransportConfig config) => RestGopeedTransport(config);

class RestGopeedTransport implements GopeedTransport {
  RestGopeedTransport(GopeedTransportConfig config)
    : _dio = Dio(
        BaseOptions(
          // Keep requests relative for deployments under a reverse proxy path.
          baseUrl: webTransportBaseUrl(config.address, Uri.base),
          contentType: Headers.jsonContentType,
          sendTimeout: const Duration(seconds: 5),
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 120),
        ),
      ) {
    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (config.apiToken.isNotEmpty) {
            options.headers['X-Api-Token'] = config.apiToken;
          }
          handler.next(options);
        },
        onError: (error, handler) {
          if (error.response?.statusCode == 401) {
            config.onUnauthorized?.call();
          }
          handler.next(error);
        },
      ),
    );
  }

  final Dio _dio;

  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    final response = await _dio.request<dynamic>(
      path,
      data: data,
      queryParameters: queryParameters,
      options: Options(method: method),
    );
    return response.data;
  }

  @override
  String join(String path) {
    final baseUrl = _dio.options.baseUrl;
    final cleanBaseUrl = baseUrl.endsWith('/') ? baseUrl.substring(0, baseUrl.length - 1) : baseUrl;
    final cleanPath = path.startsWith('/') ? path.substring(1) : path;
    return '$cleanBaseUrl/$cleanPath';
  }
}
