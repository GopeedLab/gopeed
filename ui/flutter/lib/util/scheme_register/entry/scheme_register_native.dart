import 'dart:io';

import 'package:win32_registry/win32_registry.dart';

import '../../util.dart';

doRegisterUrlScheme(String scheme) {
  if (Util.isWindows()) {
    final schemeKey = 'Software\\Classes\\$scheme';
    final appPath = Platform.resolvedExecutable;

    _upsertRegistry(
      schemeKey,
      'URL Protocol',
      '',
    );
    _upsertRegistry(
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
    _upsertRegistry(
      _torrentRegKey,
      '',
      _torrentRegValue,
    );
    _upsertRegistry(
      _torrentAppRegKey,
      '',
      'Torrent file',
    );
    _upsertRegistry(
      '$_torrentAppRegKey\\DefaultIcon',
      '',
      iconPath,
    );
    _upsertRegistry(
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

// Add Windows registry key and value if not exists
_upsertRegistry(String keyPath, String valueName, String value) {
  RegistryKey regKey;
  try {
    regKey = Registry.openPath(RegistryHive.currentUser,
        path: keyPath, desiredAccessRights: AccessRights.allAccess);
  } catch (e) {
    regKey = Registry.currentUser.createKey(keyPath);
  }

  if (regKey.getValueAsString(valueName) != value) {
    regKey
        .createValue(RegistryValue(valueName, RegistryValueType.string, value));
  }
  regKey.close();
}
