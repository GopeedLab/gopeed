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
    if (Platform.isWindows || Platform.isMacOS || Platform.isLinux) {
      await OpenDir().openNativeDir(path: directoryPath);
    } else {
      await launchUrlString("file://$directoryPath");
    }
  }

  static Future<void> _openFile(String filePath) async {
    if (Platform.isWindows || Platform.isMacOS || Platform.isLinux) {
      final fileName = path.basename(filePath);
      final parentPath = path.dirname(filePath);
      await OpenDir()
          .openNativeDir(path: parentPath, highlightedFileName: fileName);
    }
  }
}
