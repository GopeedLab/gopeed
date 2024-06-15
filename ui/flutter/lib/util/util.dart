import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'package:path/path.dart' as path;

class Util {
  static String? _storageDir;

  static String cleanPath(String path) {
    path = path.replaceAll(RegExp(r'\\'), "/");
    if (path.startsWith(".")) {
      path = path.substring(1);
    }
    if (path.startsWith("/")) {
      path = path.substring(1);
    }
    return path;
  }

  static String safeDir(String path) {
    if (path == "." || path == "./" || path == ".\\") {
      return "";
    }
    return path;
  }

  static String safePathJoin(List<String> paths) {
    return paths
        .where((e) => e.isNotEmpty)
        .map((e) => safeDir(e))
        .join("/")
        .replaceAll(RegExp(r'//'), "/");
  }

  static String fmtByte(int byte) {
    if (byte < 0) {
      return "0 B";
    } else if (byte < 1024) {
      return "$byte B";
    } else if (byte < 1024 * 1024) {
      return "${(byte / 1024).toStringAsFixed(2)} KB";
    } else if (byte < 1024 * 1024 * 1024) {
      return "${(byte / 1024 / 1024).toStringAsFixed(2)} MB";
    } else {
      return "${(byte / 1024 / 1024 / 1024).toStringAsFixed(2)} GB";
    }
  }

  static Future<void> initStorageDir() async {
    var storageDir = "";
    if (Util.isWindows()) {
      storageDir = File(Platform.resolvedExecutable).parent.path;
    } else if (!Util.isWeb()) {
      if (Util.isLinux()) {
        storageDir = File(Platform.resolvedExecutable).parent.path;
        // check has write permission, if not, fallback to application support dir
        try {
          final testFile = File(path.join(storageDir, ".test"));
          await testFile.writeAsString("test");
          await testFile.delete();
        } catch (e) {
          storageDir = (await getApplicationSupportDirectory()).path;
        }
      } else {
        storageDir = (await getApplicationSupportDirectory()).path;
      }
    }
    _storageDir = storageDir;
  }

  static String getStorageDir() {
    return _storageDir!;
  }

  static isAndroid() {
    return !kIsWeb && Platform.isAndroid;
  }

  static isIOS() {
    return !kIsWeb && Platform.isIOS;
  }

  static isMobile() {
    return !kIsWeb && (Platform.isAndroid || Platform.isIOS);
  }

  static isDesktop() {
    if (kIsWeb) {
      return false;
    }
    return Platform.isWindows || Platform.isLinux || Platform.isMacOS;
  }

  static isWindows() {
    return !kIsWeb && Platform.isWindows;
  }

  static isMacos() {
    return !kIsWeb && Platform.isMacOS;
  }

  static isLinux() {
    return !kIsWeb && Platform.isLinux;
  }

  static isWeb() {
    return kIsWeb;
  }

  static supportUnixSocket() {
    if (kIsWeb) {
      return false;
    }
    return Platform.isLinux || Platform.isMacOS || Platform.isAndroid;
  }

  static List<String> textToLines(String text) {
    if (text.isEmpty) {
      return [];
    }
    const ls = LineSplitter();
    return ls.convert(text).where((line) => line.isNotEmpty).toList();
  }

  // if one future complete, return the result, only all future error, return the last error
  static anyOk<T>(Iterable<Future<T>> futures) {
    final completer = Completer<T>();
    Object lastError;
    var count = futures.length;
    for (var future in futures) {
      future.then((value) {
        if (!completer.isCompleted) {
          completer.complete(value);
        }
      }).catchError((e) {
        lastError = e;
        count--;
        if (count == 0) {
          completer.completeError(lastError);
        }
      });
    }
    return completer.future;
  }

  static void Function() debounce(Function() fn, int ms) {
    Timer? timer;
    return () {
      timer?.cancel();
      timer = Timer(Duration(milliseconds: ms), fn);
    };
  }
}
