import 'dart:io';

import 'package:win32_registry/win32_registry.dart';

import '../../util.dart';
import '../../win32.dart';

doRegisterUrlScheme(String scheme) {
  if (Util.isWindows()) {
    final schemeKey = 'Software\\Classes\\$scheme';
    final appPath = Platform.resolvedExecutable;

    upsertRegistry(
      schemeKey,
      'URL Protocol',
      '',
    );
    upsertRegistry(
      '$schemeKey\\shell\\open\\command',
      '',
      '"$appPath" "%1"',
    );
  }
}

doUnregisterUrlScheme(String scheme) {
  if (Util.isWindows()) {
    Registry.currentUser
        .deleteKey('Software\\Classes\\$scheme', recursive: true);
  }
}

const _torrentRegKey = 'Software\\Classes\\.torrent';
const _torrentRegValue = 'Gopeed_torrent';
const _torrentAppRegKey = 'Software\\Classes\\$_torrentRegValue';

/// Register as the system's default torrent client
/// 1. Register the scheme "magnet"
/// 2. Register the file type ".torrent"
doRegisterDefaultTorrentClient() {
  if (Util.isWindows()) {
    doRegisterUrlScheme("magnet");

    final appPath = Platform.resolvedExecutable;
    final iconPath =
        '${File(appPath).parent.path}\\data\\flutter_assets\\assets\\tray_icon\\icon.ico';
    upsertRegistry(
      _torrentRegKey,
      '',
      _torrentRegValue,
    );
    upsertRegistry(
      _torrentAppRegKey,
      '',
      'Torrent file',
    );
    upsertRegistry(
      '$_torrentAppRegKey\\DefaultIcon',
      '',
      iconPath,
    );
    upsertRegistry(
      '$_torrentAppRegKey\\shell\\open\\command',
      '',
      '"$appPath" "file:///%1"',
    );
  }
}

doUnregisterDefaultTorrentClient() {
  if (Util.isWindows()) {
    doUnregisterUrlScheme("magnet");

    Registry.currentUser.deleteKey(_torrentRegKey, recursive: true);
    Registry.currentUser.deleteKey(_torrentAppRegKey, recursive: true);
  }
}
