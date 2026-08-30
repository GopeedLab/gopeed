import 'dart:convert';

import 'package:dio/dio.dart';

import '../../../libgopeed_boot.dart';
import '../../../ffi/libgopeed_worker.dart';
import '../../../../util/util.dart';
import '../gopeed_transport.dart';

GopeedTransport createTransport(GopeedTransportConfig config) => NativeGopeedTransport();

class NativeGopeedTransport implements GopeedTransport {
  NativeGopeedTransport()
    : _externalDio = Dio(
        BaseOptions(
          contentType: Headers.jsonContentType,
          sendTimeout: const Duration(seconds: 5),
          connectTimeout: const Duration(seconds: 5),
          receiveTimeout: const Duration(seconds: 120),
        ),
      );

  final Dio _externalDio;
  final LibgopeedWorker? _ffiWorker = Util.isDesktop() ? LibgopeedWorker() : null;

  @override
  Future<dynamic> request(
    String path, {
    String method = 'GET',
    dynamic data,
    Map<String, dynamic>? queryParameters,
  }) async {
    final request = _RequestParts.from(path, queryParameters);
    final requestBody = data == null ? '' : jsonEncode(data);
    final payload = _ffiWorker != null
        ? await _ffiWorker.invoke(method.toUpperCase(), request.path, request.query, requestBody)
        : await LibgopeedBoot.instance.invoke(
            method.toUpperCase(),
            request.path,
            query: request.query,
            body: requestBody,
          );
    return jsonDecode(payload);
  }

  @override
  Future<Response<String>> proxyRequest(String uri, {dynamic data, Options? options}) {
    return _externalDio.request<String>(
      uri,
      data: data,
      options: (options ?? Options()).copyWith(responseType: ResponseType.plain),
    );
  }

  @override
  String join(String path) => path;
}

class _RequestParts {
  const _RequestParts(this.path, this.query);

  final String path;
  final String query;

  factory _RequestParts.from(String rawPath, Map<String, dynamic>? queryParameters) {
    var path = rawPath;
    var query = '';
    final queryIndex = rawPath.indexOf('?');
    if (queryIndex >= 0) {
      path = rawPath.substring(0, queryIndex);
      query = rawPath.substring(queryIndex + 1);
    }
    final extraQuery = _encodeQueryParameters(queryParameters);
    if (extraQuery.isNotEmpty) {
      query = query.isEmpty ? extraQuery : '$query&$extraQuery';
    }
    return _RequestParts(path, query);
  }

  static String _encodeQueryParameters(Map<String, dynamic>? queryParameters) {
    if (queryParameters == null || queryParameters.isEmpty) return '';
    final segments = <String>[];
    queryParameters.forEach((key, value) {
      if (value == null) return;
      final values = value is Iterable ? value : [value];
      for (final item in values) {
        if (item == null) continue;
        segments.add('${Uri.encodeQueryComponent(key)}=${Uri.encodeQueryComponent(item.toString())}');
      }
    });
    return segments.join('&');
  }
}
