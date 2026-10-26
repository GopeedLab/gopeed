import 'package:dio/dio.dart';

import 'model/store_extension.dart';

class GopeedSiteApi {
  GopeedSiteApi({Dio? client})
    : _client =
          client ??
          Dio(BaseOptions(connectTimeout: const Duration(seconds: 10), receiveTimeout: const Duration(seconds: 10)));

  static final instance = GopeedSiteApi();
  final Dio _client;
  final _storePages = <String, ({StoreExtensionPage page, DateTime expires})>{};
  final _storeRequests = <String, Future<StoreExtensionPage>>{};

  Future<StoreExtensionPage> getExtensions({
    int page = 1,
    int limit = 20,
    StoreExtensionSort sort = StoreExtensionSort.stars,
    StoreSortOrder order = StoreSortOrder.desc,
    String? query,
    bool forceRefresh = false,
  }) async {
    final params = {
      'view': 'summary',
      'page': page.toString(),
      'limit': limit.clamp(1, 100).toString(),
      'sort': sort.name,
      'order': order.name,
      if (query != null && query.trim().isNotEmpty) 'q': query.trim(),
    };
    final key = Uri(queryParameters: params).toString();
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
    final response = await _client.get<Map<String, dynamic>>(
      'https://gopeed.com/api/extensions',
      queryParameters: params,
    );
    return StoreExtensionPage.fromJson(response.data!);
  }

  Future<StoreExtension> getExtension(String id, {String? version}) async {
    final uri = Uri(
      scheme: 'https',
      host: 'gopeed.com',
      pathSegments: ['api', 'extensions', id],
      queryParameters: {'version': ?version},
    );
    final response = await _client.get<Map<String, dynamic>>(uri.toString());
    return StoreExtension.fromJson(response.data!);
  }

  Future<dynamic> getReleases() async {
    final response = await _client.get<dynamic>(
      'https://gopeed.com/api/releases',
      queryParameters: {'per_page': 10},
      options: Options(followRedirects: false),
    );
    return response.data;
  }

  Future<void> reportExtensionInstall(String id) async {
    await _client.post<void>('https://gopeed.com/api/extensions/install', data: {'id': id});
  }
}
