import 'dart:io';

import 'package:path/path.dart' as path;
import 'package:win32_registry/win32_registry.dart';

import '../../util.dart';
import '../../win32.dart';

const _linuxDesktopFileName = 'com.gopeed.Gopeed.desktop';

void doRegisterUrlScheme(String scheme) {
  if (Util.isWindows()) {
    final schemeKey = 'Software\\Classes\\$scheme';
    final appPath = Platform.resolvedExecutable;

    upsertRegistry(schemeKey, 'URL Protocol', '');
    upsertRegistry('$schemeKey\\shell\\open\\command', '', '"$appPath" "%1"');
    return;
  }

  if (Util.isLinux()) {
    _installLinuxDesktopEntry(mimeTypes: {'x-scheme-handler/$scheme'});
    _setLinuxDefaultMime('x-scheme-handler/$scheme');
  }
}

void doUnregisterUrlScheme(String scheme) {
  if (Util.isWindows()) {
    Registry.currentUser.deleteKey('Software\\Classes\\$scheme', recursive: true);
    return;
  }

  if (Util.isLinux()) {
    return;
  }
}

const _torrentRegKey = 'Software\\Classes\\.torrent';
const _torrentRegValue = 'Gopeed_torrent';
const _torrentAppRegKey = 'Software\\Classes\\$_torrentRegValue';

/// Register as the system's default torrent client
/// 1. Register the scheme "magnet"
/// 2. Register the file type ".torrent"
void doRegisterDefaultTorrentClient() {
  if (Util.isWindows()) {
    doRegisterUrlScheme("magnet");

    final appPath = Platform.resolvedExecutable;
    final iconPath = '${File(appPath).parent.path}\\data\\flutter_assets\\assets\\tray_icon\\icon.ico';
    upsertRegistry(_torrentRegKey, '', _torrentRegValue);
    upsertRegistry(_torrentAppRegKey, '', 'Torrent file');
    upsertRegistry('$_torrentAppRegKey\\DefaultIcon', '', iconPath);
    upsertRegistry('$_torrentAppRegKey\\shell\\open\\command', '', '"$appPath" "file:///%1"');
    return;
  }

  if (Util.isLinux()) {
    _installLinuxDesktopEntry(mimeTypes: {'x-scheme-handler/magnet', 'application/x-bittorrent'});
    _setLinuxDefaultMime('x-scheme-handler/magnet');
    _setLinuxDefaultMime('application/x-bittorrent');
  }
}

void doUnregisterDefaultTorrentClient() {
  if (Util.isWindows()) {
    doUnregisterUrlScheme("magnet");

    Registry.currentUser.deleteKey(_torrentRegKey, recursive: true);
    Registry.currentUser.deleteKey(_torrentAppRegKey, recursive: true);
    return;
  }

  if (Util.isLinux()) {
    return;
  }
}

void _installLinuxDesktopEntry({required Set<String> mimeTypes}) {
  final applicationsDir = path.join(Platform.environment['HOME'] ?? '', '.local', 'share', 'applications');
  if (applicationsDir.startsWith('.local')) {
    return;
  }

  final desktopFile = File(path.join(applicationsDir, _linuxDesktopFileName));
  final existingMimeTypes = _readLinuxDesktopMimeTypes(desktopFile);
  final allMimeTypes = <String>{'x-scheme-handler/gopeed', ...existingMimeTypes, ...mimeTypes};

  desktopFile.parent.createSync(recursive: true);
  desktopFile.writeAsStringSync('''
[Desktop Entry]
Name=Gopeed
GenericName=Download Manager
Comment=A modern download manager for all platforms
Terminal=false
Exec=${Platform.resolvedExecutable} %U
Icon=com.gopeed.Gopeed
Type=Application
Categories=Utility;Network;
Keywords=Bittorrent;Downloader;
MimeType=${allMimeTypes.join(';')};
''');

  Process.runSync('update-desktop-database', [applicationsDir]);
}

Set<String> _readLinuxDesktopMimeTypes(File desktopFile) {
  if (!desktopFile.existsSync()) {
    return {};
  }

  for (final line in desktopFile.readAsLinesSync()) {
    if (line.startsWith('MimeType=')) {
      return line
          .substring('MimeType='.length)
          .split(';')
          .map((value) => value.trim())
          .where((value) => value.isNotEmpty)
          .toSet();
    }
  }

  return {};
}

void _setLinuxDefaultMime(String mimeType) {
  Process.runSync('xdg-mime', ['default', _linuxDesktopFileName, mimeType]);
}
