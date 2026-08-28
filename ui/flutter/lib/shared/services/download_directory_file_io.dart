import 'dart:io';

import 'package:path/path.dart' as path;

Future<void> verifyDownloadDirectoryWritable(String directoryPath) async {
  final probe = File(path.join(directoryPath, '.gopeed-write-test-${DateTime.now().microsecondsSinceEpoch}.tmp'));
  try {
    await probe.create(recursive: true);
    await probe.writeAsString('test', flush: true);
  } finally {
    if (await probe.exists()) await probe.delete();
  }
}
