import '../../database/database.dart';
import '../../util/util.dart';

class AppInitializer {
  static Future<void>? _storageInitialization;
  static Future<void>? _databaseInitialization;

  static Future<void> ensureStorageInitialized() {
    return _storageInitialization ??= Util.initStorageDir();
  }

  static Future<void> ensureDatabaseInitialized() {
    return _databaseInitialization ??= _initializeDatabase();
  }

  static Future<void> _initializeDatabase() async {
    await ensureStorageInitialized();
    await Database.instance.init();
  }
}
