import 'dart:io';

class Util {
  static String fmtByte(int byte) {
    if (byte < 1024) {
      return "$byte B";
    } else if (byte < 1024 * 1024) {
      return "${(byte / 1024).toStringAsFixed(2)} KB";
    } else if (byte < 1024 * 1024 * 1024) {
      return "${(byte / 1024 / 1024).toStringAsFixed(2)} MB";
    } else {
      return "${(byte / 1024 / 1024 / 1024).toStringAsFixed(2)} GB";
    }
  }

  static String buildPath(String path, String name) {
    return (path == "" ? "" : "$path/") + name;
  }

  static String buildAbsPath(String dir, String path, String name) {
    return (dir == "" ? "" : "$dir/") + buildPath(path, name);
  }

  static isDesktop() {
    return Platform.isWindows || Platform.isLinux || Platform.isMacOS;
  }

  static isUnix() {
    return !Platform.isWindows;
  }
}
