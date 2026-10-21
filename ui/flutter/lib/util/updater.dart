import 'dart:io';

import 'package:install_plugin/install_plugin.dart';
import 'package:path/path.dart' as path;
import 'package:path_provider/path_provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../core/capabilities/gopeed_capability.dart';
import '../core/network/candidate_download.dart';
import '../core/network/app_update_client.dart';
import 'arch/arch.dart';
import 'log_util.dart';
import 'package_info.dart';
import 'util.dart';

export '../core/network/app_update_client.dart' show VersionInfo;

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

Future<VersionInfo?> checkUpdate() => AppUpdateClient().check(appVersion);

/// Package selection belongs to the client, including when the Downloader is remote.
String appUpdateAssetName(String version, UpdateChannel? channel, String architecture) {
  final arch = {'ia32': '386', 'x64': 'amd64'}[architecture] ?? architecture;
  final prefix = 'Gopeed-v$version';
  return switch (channel) {
    UpdateChannel.windowsInstaller => '$prefix-windows-$arch.zip',
    UpdateChannel.windowsPortable => '$prefix-windows-$arch-portable.zip',
    UpdateChannel.macosDmg => '$prefix-macos-$arch.dmg',
    UpdateChannel.linuxDeb => '$prefix-linux-$arch.deb',
    UpdateChannel.androidApk => switch (architecture) {
      'arm' => '$prefix-android-armeabi-v7a.apk',
      'arm64' => '$prefix-android-arm64-v8a.apk',
      'x64' => '$prefix-android-x86_64.apk',
      _ => '$prefix-android.apk',
    },
    _ => '',
  };
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
  required GopeedService service,
  required UpdateProgressCallback onProgress,
}) async {
  final channel = updateChannel;
  final assetName = appUpdateAssetName(versionInfo.version, channel, getArch().name);
  var assetPath = '';

  if (assetName.isNotEmpty) {
    final expectedSha256 = versionInfo.assetSha256[assetName];
    if (expectedSha256 == null) throw const DownloadChecksumException();
    onProgress(0, 0);
    assetPath = path.join((await getTemporaryDirectory()).path, assetName);
    final urls = await service.getMirrorCandidates(
      'github',
      'https://github.com/GopeedLab/gopeed/releases/download/v${versionInfo.version}/$assetName',
    );
    await downloadFromCandidates(urls, assetPath, expectedSha256: expectedSha256, onReceiveProgress: onProgress);
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
