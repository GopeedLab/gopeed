import '../core/capabilities/app_capabilities.dart';
import '../core/capabilities/gopeed_capability.dart';
import 'model/store_extension.dart';

class GopeedSiteApi {
  GopeedSiteApi(this._service);

  static final instance = GopeedSiteApi(LocalAppCapabilities.instance.capabilities.gopeed);
  final GopeedService _service;
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
    final json = await _service.getStoreExtensions(params);
    return StoreExtensionPage.fromJson(json);
  }

  Future<StoreExtension> getExtension(String id, {String? version}) async {
    return StoreExtension.fromJson(await _service.getStoreExtension(id, version: version));
  }

  Future<void> reportExtensionInstall(String id) => _service.reportStoreInstall(id);
}
