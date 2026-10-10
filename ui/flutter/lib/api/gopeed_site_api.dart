import 'dart:convert';

import 'package:dio/dio.dart';

import 'api.dart';
import 'model/store_extension.dart';

class GopeedSiteApi {
  GopeedSiteApi._();

  static final instance = GopeedSiteApi._();

  static const _host = 'gopeed.com';
  final _storePages = <String, ({StoreExtensionPage page, DateTime expires})>{};
  final _storeRequests = <String, Future<StoreExtensionPage>>{};

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
    bool forceRefresh = false,
  }) async {
    final params = {
      'page': page.toString(),
      'limit': limit.clamp(1, 100).toString(),
      'sort': sort.name,
      'order': order.name,
      'view': 'summary',
      if (query != null && query.trim().isNotEmpty) 'q': query.trim(),
    };
    final key = Uri.https(_host, '/api/extensions', params).toString();
    final now = DateTime.now();
    _storePages.removeWhere((_, entry) => !now.isBefore(entry.expires));
    final cached = _storePages[key];
    if (!forceRefresh && cached != null) return cached.page;
    final pending = _storeRequests[key];
    if (pending != null) return pending;
    final request = _fetchStorePage(params);
    _storeRequests[key] = request;
    try {
      final result = await request;
      if (_storePages.length >= 32) _storePages.remove(_storePages.keys.first);
      _storePages[key] = (page: result, expires: DateTime.now().add(const Duration(minutes: 2)));
      return result;
    } finally {
      _storeRequests.remove(key);
    }
  }

  Future<StoreExtensionPage> _fetchStorePage(Map<String, String> params) async {
    final json = await _getJson('/api/extensions', queryParameters: params);
    return StoreExtensionPage.fromJson(json as Map<String, dynamic>);
  }

  Future<StoreExtension> getExtension(String id, {String? version}) async {
    final json = await _getJsonUri(
      Uri(
        scheme: 'https',
        host: _host,
        pathSegments: ['api', 'extensions', id],
        queryParameters: {if (version != null && version.isNotEmpty) 'version': version},
      ),
    );
    return StoreExtension.fromJson(json as Map<String, dynamic>);
  }

  Future<void> reportExtensionInstall(String id) async {
    final uri = Uri.https(_host, '/api/extensions/install');
    await proxyRequest(
      uri.toString(),
      data: {'id': id},
      options: Options(method: 'POST', contentType: Headers.jsonContentType),
    );
  }

  Future<dynamic> _getJson(String path, {Map<String, String>? queryParameters}) {
    return _getJsonUri(Uri.https(_host, path, queryParameters));
  }

  Future<dynamic> _getJsonUri(Uri uri) async {
    final Response<String> response = await proxyRequest(uri.toString());
    if (response.data == null || response.data!.isEmpty) {
      throw Exception('Empty response from $uri');
    }
    return jsonDecode(response.data!);
  }
}
