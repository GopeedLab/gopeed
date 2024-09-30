import 'dart:io';

import 'package:url_launcher/url_launcher_string.dart';

class FileExplorer {
  static Future<void> openAndSelectFile(String filePath) async {
    if (await FileSystemEntity.isFile(filePath)) {
      _openFile(filePath);
    } else if (await FileSystemEntity.isDirectory(filePath)) {
      _openDirectory(filePath);
    }
  }

  static Future<void> _openDirectory(String directoryPath) async {
    await launchUrlString("file://$directoryPath");
  }

  static Future<void> _openFile(String filePath) async {
    if (Platform.isWindows) {
      Process.run('explorer.exe', ['/select,', filePath]);
    } else if (Platform.isMacOS) {
      Process.run('open', ['-R', filePath]);
    } else if (Platform.isLinux) {
      _linuxOpen(filePath);
    }
  }

  static Future<void> _linuxOpen(String filePath) async {
    if (await Process.run('which', ['xdg-open'])
        .then((value) => value.exitCode == 0)) {
      final result = await Process.run('xdg-open', [filePath]);
      if (result.exitCode != 0) {
        _openWithFileManager(filePath);
      }
    } else {
      _openWithFileManager(filePath);
    }
  }

  static Future<void> _openWithFileManager(String filePath) async {
    final desktop = Platform.environment['XDG_CURRENT_DESKTOP'];
    if (desktop == null) {
      throw Exception('XDG_CURRENT_DESKTOP is not set');
    }
    if (desktop == 'GNOME') {
      await Process.run('nautilus', ['--select', filePath]);
    } else if (desktop == 'KDE') {
      await Process.run('dolphin', ['--select', filePath]);
    } else {
      throw Exception('Unsupported desktop environment');
    }
  }
}
