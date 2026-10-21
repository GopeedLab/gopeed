import 'package:dio/dio.dart';
import 'package:pub_semver/pub_semver.dart';

import '../../api/gopeed_site_api.dart';

class VersionInfo {
  const VersionInfo({
    required this.version,
    required this.changeLog,
    required this.releaseUrl,
    this.assetSha256 = const {},
  });

  final String version;
  final String changeLog;
  final String releaseUrl;
  // Populated only from official release metadata, never from a download mirror.
  final Map<String, String> assetSha256;
}

/// Query the official GitHub API directly, then the independent website API
/// only if the GitHub request or its response validation fails.
class AppUpdateClient {
  AppUpdateClient({Dio? client, GopeedSiteApi? site})
    : _client =
          client ??
          Dio(BaseOptions(connectTimeout: const Duration(seconds: 10), receiveTimeout: const Duration(seconds: 30))),
      _site = site ?? GopeedSiteApi.instance;

  static const releasesUrl = 'https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=10';
  final Dio _client;
  final GopeedSiteApi _site;

  Future<List<_Release>> _fetchGithubReleases() async {
    final cancel = CancelToken();
    try {
      final response = await _client
          .get<dynamic>(releasesUrl, cancelToken: cancel, options: Options(followRedirects: false))
          .timeout(const Duration(seconds: 30));
      return _decodeReleases(response.data);
    } finally {
      cancel.cancel('GitHub release request completed');
    }
  }

  Future<VersionInfo?> check(String currentVersion) async {
    final current = _parseVersion(currentVersion);
    if (current == null) throw FormatException('Invalid current application version', currentVersion);
    List<_Release> releases;
    try {
      releases = await _fetchGithubReleases();
    } catch (_) {
      releases = _decodeReleases(await _site.getReleases());
    }
    _Release? selected;
    for (final release in releases) {
      final version = release.version;
      if (release.draft || version.compareTo(current) <= 0) continue;
      final preview = release.prerelease || version.isPreRelease;
      if (!current.isPreRelease && preview) continue;
      if (current.isPreRelease &&
          preview &&
          (version.major != current.major || version.minor != current.minor || version.patch != current.patch)) {
        continue;
      }
      if (selected == null || version.compareTo(selected.version) > 0) selected = release;
    }
    if (selected == null) return null;
    return VersionInfo(
      version: selected.tag.replaceFirst(RegExp(r'^v'), ''),
      changeLog: selected.body,
      assetSha256: Map.unmodifiable(selected.assetSha256),
      releaseUrl: selected.url.isEmpty
          ? 'https://github.com/GopeedLab/gopeed/releases/tag/${selected.tag}'
          : selected.url,
    );
  }
}

// Build metadata does not affect SemVer precedence.
Version? _parseVersion(String raw) {
  try {
    final version = Version.parse(raw.replaceFirst(RegExp(r'^v'), ''));
    return Version(version.major, version.minor, version.patch, pre: version.preRelease.join('.'));
  } on FormatException {
    return null;
  }
}

List<_Release> _decodeReleases(dynamic data) {
  if (data is! List) throw const FormatException('Invalid release list');
  final releases = <_Release>[];
  for (final entry in data) {
    if (entry is! Map || entry['tag_name'] is! String) continue;
    final tag = entry['tag_name'] as String;
    final version = _parseVersion(tag);
    if (version == null) continue;
    releases.add(
      _Release(
        tag: tag,
        version: version,
        draft: entry['draft'] == true,
        prerelease: entry['prerelease'] == true,
        body: entry['body'] as String? ?? '',
        url: entry['html_url'] as String? ?? '',
        assetSha256: _decodeAssetSha256(entry['assets']),
      ),
    );
  }
  if (data.isNotEmpty && releases.isEmpty) throw const FormatException('Response contains no valid releases');
  return releases;
}

Map<String, String> _decodeAssetSha256(dynamic assets) {
  if (assets == null) return {};
  if (assets is! List) throw const FormatException('Invalid release assets');
  final result = <String, String>{};
  for (final asset in assets) {
    if (asset is! Map || asset['name'] is! String || asset['digest'] is! String) continue;
    final digest = asset['digest'] as String;
    final match = RegExp(r'^sha256:([a-fA-F0-9]{64})$').firstMatch(digest);
    if (match == null || match.end != digest.length) continue;
    final name = asset['name'] as String;
    if (result.containsKey(name)) throw const FormatException('Duplicate release asset');
    result[name] = match.group(1)!.toLowerCase();
  }
  return result;
}

class _Release {
  const _Release({
    required this.tag,
    required this.version,
    required this.draft,
    required this.prerelease,
    required this.body,
    required this.url,
    required this.assetSha256,
  });
  final String tag;
  final Version version;
  final bool draft;
  final bool prerelease;
  final String body;
  final String url;
  final Map<String, String> assetSha256;
}
