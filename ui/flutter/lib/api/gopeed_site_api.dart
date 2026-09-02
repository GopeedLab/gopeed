import 'dart:convert';

import 'package:dio/dio.dart';

import 'api.dart';
import 'model/store_extension.dart';

class GopeedSiteApi {
  GopeedSiteApi._();

  static final instance = GopeedSiteApi._();

  static const _host = 'gopeed.com';

  Future<List<dynamic>> getReleases({int perPage = 10}) async {
    final json = await _getJson('/api/releases', queryParameters: {'per_page': perPage.clamp(1, 100).toString()});
    if (json is! List<dynamic>) {
      throw const FormatException('Invalid releases response');
    }
    return json;
  }

  Future<StoreExtensionPage> getExtensions({
    int page = 1,
    int limit = 20,
    StoreExtensionSort sort = StoreExtensionSort.stars,
    StoreSortOrder order = StoreSortOrder.desc,
    String? query,
  }) async {
    final json = await _getJson(
      '/api/extensions',
      queryParameters: {
        'page': page.toString(),
        'limit': limit.clamp(1, 100).toString(),
        'sort': sort.name,
        'order': order.name,
        if (query != null && query.trim().isNotEmpty) 'q': query.trim(),
      },
    );
    return StoreExtensionPage.fromJson(json as Map<String, dynamic>);
  }

  Future<void> reportExtensionInstall(String id) async {
    final uri = Uri.https(_host, '/api/extensions/install');
    await proxyRequest(
      uri.toString(),
      data: {'id': id},
      options: Options(method: 'POST', contentType: Headers.jsonContentType),
    );
  }

  Future<dynamic> _getJson(String path, {Map<String, String>? queryParameters}) async {
    final uri = Uri.https(_host, path, queryParameters);
    final Response<String> response = await proxyRequest(uri.toString());
    if (response.data == null || response.data!.isEmpty) {
      throw Exception('Empty response from $uri');
    }
    return jsonDecode(response.data!);
  }
}
