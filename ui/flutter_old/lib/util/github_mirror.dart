import 'package:dio/dio.dart';
import 'package:get/get.dart';

import '../api/model/downloader_config.dart';
import '../app/modules/app/controllers/app_controller.dart';

enum MirrorType {
  githubSource,
  githubRelease,
}

/// Get the configured mirrors
List<GithubMirror> _getConfiguredMirrors() {
  try {
    final appController = Get.find<AppController>();
    final config = appController.downloaderConfig.value.extra.githubMirror;

    if (!config.enabled) {
      return [];
    }

    // Return configured mirrors (filtering not deleted ones)
    return config.mirrors.where((m) => !m.isDeleted).toList();
  } catch (e) {
    // Fallback to empty list if controller not found
    return [];
  }
}

/// Auto detect the best mirror for the given [rawUrl] and [type]
Future<String> githubAutoMirror(String rawUrl, MirrorType type) async {
  final mirrorUrls = githubMirrorUrls(rawUrl, type);

  // If no mirrors, return original URL
  if (mirrorUrls.isEmpty) {
    return rawUrl;
  }

  // Ping all mirrors and get the fastest one
  final pingResult = await Future.wait(mirrorUrls.map((e) async {
    final client = Dio()
      ..options.sendTimeout = const Duration(seconds: 3)
      ..options.connectTimeout = const Duration(seconds: 3);
    var time = DateTime.now().millisecondsSinceEpoch;
    try {
      final response = await client.head(e);
      if (response.statusCode == 200) {
        time = DateTime.now().millisecondsSinceEpoch - time;
        return (e, time);
      }
    } catch (e) {
      // ignore
    } finally {
      client.close();
    }
    return (e, -1);
  }));

  var list = pingResult.where((e) => e.$2 != -1).toList()
    ..sort((a, b) => a.$2.compareTo(b.$2));
  if (list.isNotEmpty) {
    return list.first.$1;
  }

  return rawUrl;
}

List<String> githubMirrorUrls(String rawUrl, MirrorType type) {
  final mirrors = _getConfiguredMirrors();

  final ret = <String>[];
  for (final mirror in mirrors) {
    String? mirrorUrl;

    if (mirror.type == GithubMirrorType.jsdelivr) {
      // jsdelivr only supports source files
      if (type != MirrorType.githubSource) {
        continue;
      }

      // Transform: https://raw.githubusercontent.com/user/repo/master/path
      // To: https://fastly.jsdelivr.net/gh/user/repo/path
      final jsDelivrPattern = RegExp(
          r'.*raw\.githubusercontent\.com(/[^/]+)(/[^/]+)/(?:master|main)(/.*)');
      final match = jsDelivrPattern.firstMatch(rawUrl);
      if (match != null) {
        final user = match.group(1);
        final repo = match.group(2);
        final path = match.group(3);
        mirrorUrl = '${mirror.url}$user$repo$path';
      }
    } else if (mirror.type == GithubMirrorType.ghProxy) {
      // gh-proxy supports both source and release
      // Simply prepend the mirror URL to the original URL
      mirrorUrl = '${mirror.url}/$rawUrl';
    }

    if (mirrorUrl != null) {
      ret.add(mirrorUrl);
    }
  }

  return ret;
}
