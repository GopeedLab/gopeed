import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:install_plugin/install_plugin.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
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
  String? releaseDataStr;
  try {
    releaseDataStr = (await api.proxyRequest('https://api.github.com/repos/GopeedLab/gopeed/releases/latest')).data;
  } catch (_) {
    releaseDataStr = jsonEncode(await GopeedSiteApi.instance.getRelease());
  }
  if (releaseDataStr == null || releaseDataStr.isEmpty) return null;

  final releaseData = jsonDecode(releaseDataStr) as Map<String, dynamic>;
  final tagName = releaseData['tag_name'] as String?;
  if (tagName == null || tagName.isEmpty) return null;
  final latestVersion = tagName.startsWith('v') ? tagName.substring(1) : tagName;
  if (!isNewerVersion(latestVersion, packageInfo.version)) return null;

  return VersionInfo(
    version: latestVersion,
    changeLog: (releaseData['body'] ?? '').toString(),
    releaseUrl: (releaseData['html_url'] ?? 'https://github.com/GopeedLab/gopeed/releases/latest').toString(),
  );
}

bool isNewerVersion(String latest, String current) {
  final latestParts = _versionParts(latest);
  final currentParts = _versionParts(current);
  final length = latestParts.length > currentParts.length ? latestParts.length : currentParts.length;
  for (var index = 0; index < length; index++) {
    final latestPart = index < latestParts.length ? latestParts[index] : 0;
    final currentPart = index < currentParts.length ? currentParts[index] : 0;
    if (latestPart > currentPart) return true;
    if (latestPart < currentPart) return false;
  }
  return false;
}

List<int> _versionParts(String version) {
  return version
      .split(RegExp(r'[^0-9]+'))
      .where((part) => part.isNotEmpty)
      .map((part) => int.tryParse(part) ?? 0)
      .toList();
}

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
