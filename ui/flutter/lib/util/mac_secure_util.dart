import 'dart:io';

import 'package:macos_secure_bookmarks/macos_secure_bookmarks.dart';
import 'package:shared_preferences/shared_preferences.dart';

class MacSecureUtil {
  static const _bookmarkKey = "bookmark:";

  static Future<void> saveBookmark(String dir) async {
    if (!Platform.isMacOS) {
      return;
    }
    final secureBookmarks = SecureBookmarks();
    final bookmark = await secureBookmarks.bookmark(Directory(dir));
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_bookmarkKey + dir, bookmark);
  }

  static Future<void> loadBookmark() async {
    if (!Platform.isMacOS) {
      return;
    }
    final prefs = await SharedPreferences.getInstance();
    final keys = prefs.getKeys();
    final secureBookmarks = SecureBookmarks();
    keys.where((k) => k.startsWith(_bookmarkKey)).forEach((k) async {
      final resolvedFile = await secureBookmarks
          .resolveBookmark(prefs.getString(k)!, isDirectory: true);
      await secureBookmarks.startAccessingSecurityScopedResource(resolvedFile);
    });
  }
}
