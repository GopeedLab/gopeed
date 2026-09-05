import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:install_plugin/install_plugin.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:url_launcher/url_launcher.dart';

import '../api/api.dart' as api;
import '../api/gopeed_site_api.dart';
import '../api/model/downloader_config.dart';
import 'arch/arch.dart';
import 'github_mirror.dart';
import 'log_util.dart';
import 'package_info.dart';
import 'util.dart';

enum UpdateChannel {
  windowsInstaller,
  windowsPortable,
  macosDmg,
  linuxFlathub,
  linuxSnap,
  linuxDeb,
  linuxAppImage,
  linuxRpm,
  androidApk,
  iosIpa,
  docker,
}

const _channelEnv = String.fromEnvironment('UPDATE_CHANNEL');

UpdateChannel? get updateChannel {
  for (final channel in UpdateChannel.values) {
    if (channel.name == _channelEnv) return channel;
  }
  if (Util.isWindows()) return UpdateChannel.windowsPortable;
  if (Util.isMacos()) return UpdateChannel.macosDmg;
  if (Util.isAndroid()) return UpdateChannel.androidApk;
  if (Util.isIOS()) return UpdateChannel.iosIpa;
  return null;
}

String get _updaterBinaryName => 'updater${Util.isWindows() ? '.exe' : ''}';

const _releasePageSize = 10;
const _githubReleasesUrl = 'https://api.github.com/repos/GopeedLab/gopeed/releases?per_page=$_releasePageSize';

class VersionInfo {
  const VersionInfo({required this.version, required this.changeLog, required this.releaseUrl});

  final String version;
  final String changeLog;
  final String releaseUrl;
}

/// Installs the small native updater bundled with desktop releases.
///
/// Development builds do not necessarily contain this asset. Callers should
/// treat a missing asset as recoverable; [updateApp] falls back to the release
/// page when the helper is unavailable.
Future<void> installUpdater() async {
  if (!Util.isDesktop()) return;
  await Util.installAsset(
    'assets/exec/$_updaterBinaryName',
    await Util.homePathJoin(_updaterBinaryName),
    executable: true,
  );
}

Future<VersionInfo?> checkUpdate() async {
  List<dynamic> releases;
  try {
    final releaseDataStr = (await api.proxyRequest(_githubReleasesUrl)).data;
    if (releaseDataStr == null || releaseDataStr.isEmpty) {
      throw const FormatException('Empty GitHub releases response');
    }
    final releaseData = jsonDecode(releaseDataStr);
    if (releaseData is! List<dynamic>) {
      throw const FormatException('Invalid GitHub releases response');
    }
    releases = releaseData;
  } catch (_) {
    releases = await GopeedSiteApi.instance.getReleases(perPage: _releasePageSize);
  }
  return selectUpdateRelease(releases, appVersion);
}

bool isNewerVersion(String latest, String current) {
  try {
    return _parseVersion(latest).compareTo(_parseVersion(current)) > 0;
  } on FormatException {
    return false;
  }
}

VersionInfo? selectUpdateRelease(List<dynamic> releases, String currentVersionText) {
  late final Version currentVersion;
  try {
    currentVersion = _parseVersion(currentVersionText);
  } on FormatException {
    return null;
  }

  Map<String, dynamic>? selectedRelease;
  Version? selectedVersion;

  for (final item in releases) {
    if (item is! Map) continue;
    final release = Map<String, dynamic>.from(item);
    if (release['draft'] == true) continue;

    final tagName = release['tag_name'];
    if (tagName is! String || tagName.isEmpty) continue;

    late final Version candidateVersion;
    try {
      candidateVersion = _parseVersion(tagName);
    } on FormatException {
      continue;
    }

    if (candidateVersion.compareTo(currentVersion) <= 0) continue;

    final candidateIsPrerelease = release['prerelease'] == true || candidateVersion.isPreRelease;
    if (!currentVersion.isPreRelease) {
      if (candidateIsPrerelease) continue;
    } else if (candidateIsPrerelease && !_hasSameReleaseCore(candidateVersion, currentVersion)) {
      // Preview builds graduate through their own beta/rc line before returning
      // to stable; they do not jump into the next version's preview line.
      continue;
    }

    if (selectedVersion == null || candidateVersion.compareTo(selectedVersion) > 0) {
      selectedRelease = release;
      selectedVersion = candidateVersion;
    }
  }

  if (selectedRelease == null || selectedVersion == null) return null;
  final tagName = selectedRelease['tag_name'] as String;
  return VersionInfo(
    version: _versionText(tagName),
    changeLog: (selectedRelease['body'] ?? '').toString(),
    releaseUrl: (selectedRelease['html_url'] ?? 'https://github.com/GopeedLab/gopeed/releases/tag/$tagName').toString(),
  );
}

Version _parseVersion(String version) => Version.parse(_versionText(version));

String _versionText(String version) => version.startsWith('v') ? version.substring(1) : version;

bool _hasSameReleaseCore(Version first, Version second) =>
    first.major == second.major && first.minor == second.minor && first.patch == second.patch;

/// Extracts the matching section from Gopeed's bilingual GitHub release notes.
String localizedReleaseNotes(String fullChangeLog, String languageCode) {
  final isChinese = languageCode.toLowerCase().startsWith('zh');
  final chineseStart = RegExp(r'^#\s+更新日志', multiLine: true).firstMatch(fullChangeLog)?.start;
  if (isChinese) {
    return (chineseStart == null ? fullChangeLog : fullChangeLog.substring(chineseStart)).trim();
  }
  final englishStart = RegExp(r'^#\s+Release notes', multiLine: true).firstMatch(fullChangeLog)?.start;
  if (englishStart == null) return fullChangeLog.trim();
  final englishEnd = chineseStart != null && chineseStart > englishStart ? chineseStart : fullChangeLog.length;
  return fullChangeLog.substring(englishStart, englishEnd).trim();
}

typedef UpdateProgressCallback = void Function(int received, int total);

/// Downloads and applies [versionInfo] using the build-time release channel.
/// Store-distributed and unsupported channels safely open the release page.
Future<void> updateApp(
  VersionInfo versionInfo, {
  required ExtraConfigGithubMirror githubMirror,
  required UpdateProgressCallback onProgress,
}) async {
  final channel = updateChannel;
  final assetName = updateAssetName(versionInfo.version, channel: channel);
  var assetPath = '';

  if (assetName.isNotEmpty) {
    final rawUrl = 'https://github.com/GopeedLab/gopeed/releases/download/v${versionInfo.version}/$assetName';
    assetPath = path.join((await getTemporaryDirectory()).path, assetName);
    final downloadUrl = await githubAutoMirror(rawUrl, MirrorType.githubRelease, config: githubMirror);
    final client = Dio();
    try {
      await client.download(downloadUrl, assetPath, onReceiveProgress: onProgress);
    } finally {
      client.close();
    }
  }

  switch (channel) {
    case UpdateChannel.windowsInstaller:
    case UpdateChannel.windowsPortable:
    case UpdateChannel.macosDmg:
    case UpdateChannel.linuxFlathub:
    case UpdateChannel.linuxSnap:
    case UpdateChannel.linuxDeb:
      final updaterPath = await Util.homePathJoin(_updaterBinaryName);
      if (!await File(updaterPath).exists()) {
        await _openRelease(versionInfo.releaseUrl);
        return;
      }
      await Process.run(updaterPath, [
        '-pid',
        pid.toString(),
        '-channel',
        channel!.name,
        '-asset',
        assetPath,
        '-exeDir',
        path.dirname(Platform.resolvedExecutable),
        '-log',
        path.join(logsDir(), 'updater.log'),
      ]);
    case UpdateChannel.androidApk:
      await InstallPlugin.installApk(assetPath);
    case UpdateChannel.linuxAppImage:
    case UpdateChannel.linuxRpm:
    case UpdateChannel.iosIpa:
    case UpdateChannel.docker:
    case null:
      await _openRelease(versionInfo.releaseUrl);
  }
}

Future<void> _openRelease(String releaseUrl) async {
  final opened = await launchUrl(Uri.parse(releaseUrl), mode: LaunchMode.externalApplication);
  if (!opened) throw StateError('Unable to open the release page.');
}

String updateAssetName(String version, {UpdateChannel? channel, Architecture? architecture}) {
  final targetChannel = channel ?? updateChannel;
  final arch = architecture ?? getArch();

  String commonArchName() => switch (arch) {
    Architecture.ia32 => '386',
    Architecture.x64 => 'amd64',
    _ => arch.name,
  };

  return switch (targetChannel) {
    UpdateChannel.windowsInstaller => 'Gopeed-v$version-windows-${commonArchName()}.zip',
    UpdateChannel.windowsPortable => 'Gopeed-v$version-windows-${commonArchName()}-portable.zip',
    UpdateChannel.macosDmg => 'Gopeed-v$version-macos-${commonArchName()}.dmg',
    UpdateChannel.linuxDeb => 'Gopeed-v$version-linux-${commonArchName()}.deb',
    UpdateChannel.androidApk => _androidAssetName(version, arch),
    _ => '',
  };
}

String _androidAssetName(String version, Architecture arch) {
  final archName = switch (arch) {
    Architecture.arm => 'armeabi-v7a',
    Architecture.arm64 => 'arm64-v8a',
    Architecture.x64 => 'x86_64',
    _ => null,
  };
  return 'Gopeed-v$version-android${archName == null ? '' : '-$archName'}.apk';
}
