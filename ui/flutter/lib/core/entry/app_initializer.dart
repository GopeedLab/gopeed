import '../../util/util.dart';

class AppInitializer {
  static Future<void>? _storageInitialization;

  static Future<void> ensureStorageInitialized() {
    return _storageInitialization ??= Util.initStorageDir();
  }
}
