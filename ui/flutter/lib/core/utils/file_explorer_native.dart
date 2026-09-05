import 'dart:io';

import 'package:open_dir/open_dir.dart';
import 'package:open_filex/open_filex.dart';
import 'package:path/path.dart' as path;

Future<bool> open(String filePath) async {
  try {
    if (!await FileSystemEntity.isFile(filePath) && !await FileSystemEntity.isDirectory(filePath)) {
      return false;
    }
    final result = await OpenFilex.open(filePath);
    return result.type == ResultType.done;
  } catch (_) {
    return false;
  }
}

Future<bool> reveal(String filePath) async {
  try {
    if (!(Platform.isWindows || Platform.isMacOS || Platform.isLinux)) {
      return open(filePath);
    }

    if (await FileSystemEntity.isFile(filePath)) {
      return await OpenDir().openNativeDir(
            path: path.dirname(filePath),
            highlightedFileName: path.basename(filePath),
          ) ??
          false;
    }
    if (await FileSystemEntity.isDirectory(filePath)) {
      return await OpenDir().openNativeDir(path: filePath) ?? false;
    }

    final parentPath = path.dirname(filePath);
    if (await FileSystemEntity.isDirectory(parentPath)) {
      return await OpenDir().openNativeDir(path: parentPath) ?? false;
    }
    return false;
  } catch (_) {
    return false;
  }
}
