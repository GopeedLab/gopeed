import 'package:dio/dio.dart';

const _gopeedApiBase = 'https://gopeed.com/api';

final _dio = Dio(BaseOptions(
  connectTimeout: const Duration(seconds: 5),
  receiveTimeout: const Duration(seconds: 10),
));

// ---------------------------------------------------------------------------
// Release API
// ---------------------------------------------------------------------------

/// Fetch the latest release info from gopeed.com.
/// Returns the raw JSON string of the release data, or null on failure.
Future<String?> gopeedGetRelease() async {
  final resp = await _dio.get<String>('$_gopeedApiBase/release');
  return resp.data;
}

// ---------------------------------------------------------------------------
// Extension stats API
// ---------------------------------------------------------------------------

/// Report extension installation statistics to gopeed.com.
/// This is a fire-and-forget call — errors are silently ignored.
Future<void> gopeedReportExtensionInstall(String id) async {
  try {
    await _dio.post('$_gopeedApiBase/extensions/install', data: {'id': id});
  } catch (_) {
    // Ignore errors — statistics reporting is best-effort.
  }
}

// ---------------------------------------------------------------------------
// Extension search API
// ---------------------------------------------------------------------------

enum GopeedExtensionSortField { stars, installs, updated }

enum GopeedExtensionSortOrder { asc, desc }

class GopeedExtension {
  final String id;
  final String name;
  final String author;
  final String title;
  final String description;
  final String icon;
  final String version;
  final int stars;
  final int installs;
  final String homepage;

  /// The URL used to install this extension (e.g. the GitHub repo URL).
  final String repoUrl;

  GopeedExtension({
    required this.id,
    required this.name,
    required this.author,
    required this.title,
    required this.description,
    required this.icon,
    required this.version,
    required this.stars,
    required this.installs,
    required this.homepage,
    required this.repoUrl,
  });

  factory GopeedExtension.fromJson(Map<String, dynamic> json) {
    final repo = json['repository'] as Map<String, dynamic>?;
    return GopeedExtension(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      author: json['author'] as String? ?? '',
      title: json['title'] as String? ?? '',
      description: json['description'] as String? ?? '',
      icon: _resolveIconUrl(json['icon'] as String? ?? ''),
      version: json['version'] as String? ?? '',
      stars: json['stars'] as int? ?? 0,
      installs: json['installs'] as int? ?? 0,
      homepage: json['homepage'] as String? ?? '',
      repoUrl: repo?['url'] as String? ?? '',
    );
  }

  static String _resolveIconUrl(String icon) {
    if (icon.isEmpty) return '';
    if (icon.startsWith('http')) return icon;
    if (icon.startsWith('/')) return 'https://gopeed.com$icon';
    return 'https://gopeed.com/$icon';
  }
}

class GopeedPagination {
  final int page;
  final int limit;
  final int total;
  final int totalPages;
  final bool hasNext;
  final bool hasPrev;

  GopeedPagination({
    required this.page,
    required this.limit,
    required this.total,
    required this.totalPages,
    required this.hasNext,
    required this.hasPrev,
  });

  factory GopeedPagination.fromJson(Map<String, dynamic> json) {
    return GopeedPagination(
      page: json['page'] as int? ?? 1,
      limit: json['limit'] as int? ?? 20,
      total: json['total'] as int? ?? 0,
      totalPages: json['totalPages'] as int? ?? 0,
      hasNext: json['hasNext'] as bool? ?? false,
      hasPrev: json['hasPrev'] as bool? ?? false,
    );
  }
}

class GopeedExtensionSearchResult {
  final List<GopeedExtension> data;
  final GopeedPagination pagination;

  GopeedExtensionSearchResult({
    required this.data,
    required this.pagination,
  });

  factory GopeedExtensionSearchResult.fromJson(Map<String, dynamic> json) {
    return GopeedExtensionSearchResult(
      data: (json['data'] as List? ?? [])
          .map((e) => GopeedExtension.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: GopeedPagination.fromJson(
          json['pagination'] as Map<String, dynamic>? ?? {}),
    );
  }
}

/// Search extensions from the gopeed.com store.
///
/// [page] is 1-indexed. [q] is an optional full-text search query.
Future<GopeedExtensionSearchResult> gopeedSearchExtensions({
  int page = 1,
  int limit = 20,
  GopeedExtensionSortField sort = GopeedExtensionSortField.stars,
  GopeedExtensionSortOrder order = GopeedExtensionSortOrder.desc,
  String? q,
}) async {
  final resp = await _dio.get(
    '$_gopeedApiBase/extensions',
    queryParameters: {
      'page': page,
      'limit': limit,
      'sort': sort.name,
      'order': order.name,
      if (q != null && q.isNotEmpty) 'q': q,
    },
  );
  return GopeedExtensionSearchResult.fromJson(
      resp.data as Map<String, dynamic>);
}
