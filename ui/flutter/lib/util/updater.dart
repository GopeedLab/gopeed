import 'dart:io';

import 'package:install_plugin/install_plugin.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../core/capabilities/gopeed_capability.dart';
import '../core/network/candidate_download.dart';
import 'arch/arch.dart';
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
  const VersionInfo({
    required this.version,
    required this.changeLog,
    required this.releaseUrl,
    required this.assetName,
    required this.downloadUrls,
  });

  factory VersionInfo.fromJson(Map<String, dynamic> json) => VersionInfo(
    version: json['version'] as String,
    changeLog: json['changeLog'] as String,
    releaseUrl: json['releaseUrl'] as String,
    assetName: json['assetName'] as String,
    downloadUrls: List<String>.from(json['downloadUrls'] as List),
  );

  final String version;
  final String changeLog;
  final String releaseUrl;
  final String assetName;
  final List<String> downloadUrls;
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

Future<VersionInfo?> checkUpdate({required GopeedService service}) async {
  final result = await service.checkAppUpdate(appVersion, channel: updateChannel?.name, architecture: getArch().name);
  return result == null ? null : VersionInfo.fromJson(result);
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
Future<void> updateApp(VersionInfo versionInfo, {required UpdateProgressCallback onProgress}) async {
  final channel = updateChannel;
  final assetName = versionInfo.assetName;
  var assetPath = '';

  if (assetName.isNotEmpty) {
    onProgress(0, 0);
    assetPath = path.join((await getTemporaryDirectory()).path, assetName);
    await downloadFromCandidates(versionInfo.downloadUrls, assetPath, onReceiveProgress: onProgress);
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
