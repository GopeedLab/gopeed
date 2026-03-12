import 'dart:convert';

class StoreExtensionPage {
  final List<StoreExtension> data;
  final StorePagination pagination;

  StoreExtensionPage({required this.data, required this.pagination});

  factory StoreExtensionPage.fromJson(Map<String, dynamic> json) {
    return StoreExtensionPage(
      data: (json['data'] as List<dynamic>? ?? [])
          .map((e) => StoreExtension.fromJson(e as Map<String, dynamic>))
          .toList(),
      pagination: StorePagination.fromJson(
          json['pagination'] as Map<String, dynamic>? ?? {}),
    );
  }
}

class StorePagination {
  final int page;
  final int limit;
  final int total;
  final int totalPages;
  final bool hasNext;
  final bool hasPrev;

  StorePagination({
    required this.page,
    required this.limit,
    required this.total,
    required this.totalPages,
    required this.hasNext,
    required this.hasPrev,
  });

  factory StorePagination.fromJson(Map<String, dynamic> json) {
    return StorePagination(
      page: (json['page'] as num?)?.toInt() ?? 1,
      limit: (json['limit'] as num?)?.toInt() ?? 20,
      total: (json['total'] as num?)?.toInt() ?? 0,
      totalPages: (json['totalPages'] as num?)?.toInt() ?? 1,
      hasNext: json['hasNext'] as bool? ?? false,
      hasPrev: json['hasPrev'] as bool? ?? false,
    );
  }
}

class StoreExtension {
  final String id;
  final String repoFullName;
  final String repoUrl;
  final String? directory;
  final String? commitSha;
  final String name;
  final String author;
  final String title;
  final String description;
  final String? icon;
  final String version;
  final String? homepage;
  final String? readme;
  final int installCount;
  final int stars;
  final List<String> topics;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  StoreExtension({
    required this.id,
    required this.repoFullName,
    required this.repoUrl,
    this.directory,
    this.commitSha,
    required this.name,
    required this.author,
    required this.title,
    required this.description,
    this.icon,
    required this.version,
    this.homepage,
    this.readme,
    required this.installCount,
    required this.stars,
    required this.topics,
    this.createdAt,
    this.updatedAt,
  });

  factory StoreExtension.fromJson(Map<String, dynamic> json) {
    return StoreExtension(
      id: json['id'] as String? ?? '',
      repoFullName: json['repoFullName'] as String? ?? '',
      repoUrl: json['repoUrl'] as String? ?? '',
      directory: json['directory'] as String?,
      commitSha: json['commitSha'] as String?,
      name: json['name'] as String? ?? '',
      author: json['author'] as String? ?? '',
      title: json['title'] as String? ?? '',
      description: json['description'] as String? ?? '',
      icon: json['icon'] as String?,
      version: json['version'] as String? ?? '0.0.0',
      homepage: json['homepage'] as String?,
      readme: json['readme'] as String?,
      installCount: (json['installCount'] as num?)?.toInt() ?? 0,
      stars: (json['stars'] as num?)?.toInt() ?? 0,
      topics: _parseTopics(json['topics']),
      createdAt: _parseDate(json['createdAt']),
      updatedAt: _parseDate(json['updatedAt']),
    );
  }

  static List<String> _parseTopics(dynamic value) {
    if (value is List) {
      return value.map((e) => e.toString()).toList();
    }
    if (value is String && value.isNotEmpty) {
      try {
        final parsed = jsonDecode(value);
        if (parsed is List) {
          return parsed.map((e) => e.toString()).toList();
        }
      } catch (_) {}
    }
    return const [];
  }

  static DateTime? _parseDate(dynamic value) {
    if (value == null) return null;
    if (value is int) {
      return DateTime.fromMillisecondsSinceEpoch(value);
    }
    if (value is String && value.isNotEmpty) {
      return DateTime.tryParse(value);
    }
    return null;
  }
}

enum StoreExtensionSort {
  stars,
  installs,
  updated,
}

enum StoreSortOrder {
  asc,
  desc,
}
