import 'package:dio/dio.dart';

import '../api/model/downloader_config.dart';

enum MirrorType { githubSource, githubRelease }

Future<String> githubAutoMirror(String rawUrl, MirrorType type, {required ExtraConfigGithubMirror config}) async {
  final mirrorUrls = githubMirrorUrls(rawUrl, type, config: config);
  if (mirrorUrls.isEmpty) return rawUrl;

  final results = await Future.wait(
    mirrorUrls.map((url) async {
      final client = Dio(
        BaseOptions(connectTimeout: const Duration(seconds: 3), sendTimeout: const Duration(seconds: 3)),
      );
      final startedAt = DateTime.now();
      try {
        final response = await client.head<void>(url);
        if (response.statusCode == 200) return (url, DateTime.now().difference(startedAt));
      } catch (_) {
        // A failed mirror probe must never prevent the original download.
      } finally {
        client.close();
      }
      return (url, null);
    }),
  );

  final available = results.where((result) => result.$2 != null).toList()
    ..sort((left, right) => left.$2!.compareTo(right.$2!));
  return available.isEmpty ? rawUrl : available.first.$1;
}

List<String> githubMirrorUrls(String rawUrl, MirrorType type, {required ExtraConfigGithubMirror config}) {
  if (!config.enabled) return const [];
  final urls = <String>[];
  for (final mirror in config.mirrors.where((item) => !item.isDeleted)) {
    switch (mirror.type) {
      case GithubMirrorType.jsdelivr:
        if (type != MirrorType.githubSource) continue;
        final match = RegExp(r'.*raw\.githubusercontent\.com(/[^/]+)(/[^/]+)/(?:master|main)(/.*)').firstMatch(rawUrl);
        if (match != null) {
          urls.add('${mirror.url}${match.group(1)}${match.group(2)}${match.group(3)}');
        }
      case GithubMirrorType.ghProxy:
        urls.add('${mirror.url}/$rawUrl');
    }
  }
  return urls;
}
