import 'dart:io';

import 'package:macos_secure_bookmarks/macos_secure_bookmarks.dart';

import '../database/database.dart';
import 'util.dart';

class MacSecureUtil {
  static Future<void> saveBookmark(String dir) async {
    if (!Util.isMacos()) {
      return;
    }
    final secureBookmarks = SecureBookmarks();
    final bookmark = await secureBookmarks.bookmark(Directory(dir));
    Database.instance.saveBookmark(MapEntry(dir, bookmark));
  }

  static Future<void> loadBookmark() async {
    if (!Util.isMacos()) {
      return;
    }
    final secureBookmarks = SecureBookmarks();
    final bookmark = Database.instance.getBookmark() ?? {};
    bookmark.forEach((_, v) async {
      final resolvedFile =
          await secureBookmarks.resolveBookmark(v, isDirectory: true);
      await secureBookmarks.startAccessingSecurityScopedResource(resolvedFile);
    });
  }
}
