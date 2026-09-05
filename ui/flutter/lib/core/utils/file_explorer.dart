import 'file_explorer_stub.dart' if (dart.library.io) 'file_explorer_native.dart' as implementation;

abstract final class FileExplorer {
  static Future<bool> open(String filePath) => implementation.open(filePath);

  static Future<bool> reveal(String filePath) => implementation.reveal(filePath);
}
