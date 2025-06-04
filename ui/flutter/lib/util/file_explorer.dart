import 'dart:io';
import 'package:open_dir/open_dir.dart';
import 'package:path/path.dart' as path;
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
    if (Platform.isWindows || Platform.isMacOS || Platform.isLinux) {
      final fileName = path.basename(filePath);
      final parentPath = path.dirname(filePath);
      await OpenDir()
          .openNativeDir(path: parentPath, highlightedFileName: fileName);
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
