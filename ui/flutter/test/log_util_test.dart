import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/util/log_util.dart';
import 'package:path/path.dart' as path;

void main() {
  late Directory temporaryDirectory;

  setUp(() async {
    temporaryDirectory = await Directory.systemTemp.createTemp('gopeed-log-util-');
  });

  tearDown(() async {
    await temporaryDirectory.delete(recursive: true);
  });

  test('creates the log directory when its parents do not exist', () {
    final logDirPath = path.join(temporaryDirectory.path, 'storage', 'logs');

    final logDirectory = ensureLogDirectory(logDirPath);

    expect(logDirectory.path, logDirPath);
    expect(logDirectory.existsSync(), isTrue);
  });

  test('allows the log directory to be initialized repeatedly', () {
    final logDirPath = path.join(temporaryDirectory.path, 'storage', 'logs');

    ensureLogDirectory(logDirPath);
    final logDirectory = ensureLogDirectory(logDirPath);

    expect(logDirectory.existsSync(), isTrue);
  });
}
