import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';

void main() {
  test('auto-start tasks uses the backend-owned top-level config field', () {
    final config = DownloaderConfig(autoStartTasks: true);
    final json = config.toJson();

    expect(json['autoStartTasks'], isTrue);
    expect(json['extra'], isNot(contains('autoStartTasks')));
    expect(DownloaderConfig.fromJson(json).autoStartTasks, isTrue);
  });

  test('legacy extra auto-start setting is not migrated', () {
    final json = DownloaderConfig().toJson();
    (json['extra'] as Map<String, dynamic>)['autoStartTasks'] = true;

    expect(DownloaderConfig.fromJson(json).autoStartTasks, isFalse);
  });
}
