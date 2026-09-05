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

  test('REST server settings round-trip through the Go downloader config', () {
    final config = DownloaderConfig()
      ..api = ApiServerConfig(enable: true, network: 'tcp', address: '127.0.0.1:4321', token: 'secret');

    final decoded = DownloaderConfig.fromJson(config.toJson()).api;
    expect(decoded.enable, isTrue);
    expect(decoded.network, 'tcp');
    expect(decoded.address, '127.0.0.1:4321');
    expect(decoded.token, 'secret');
  });

  test('Flutter preferences round-trip through the Go-owned extra config', () {
    final config = DownloaderConfig();
    config.extra
      ..windowState = WindowStateConfig(isMaximized: true, width: 1280, height: 720)
      ..bookmarks = {'downloads': 'D:/Downloads'}
      ..createHistory = ['https://example.com/file.zip']
      ..runAsMenubarApp = true
      ..analyticsEnabled = false
      ..analyticsClientId = 'client-id';

    final decoded = DownloaderConfig.fromJson(config.toJson()).extra;
    expect(decoded.windowState.isMaximized, isTrue);
    expect(decoded.windowState.width, 1280);
    expect(decoded.windowState.height, 720);
    expect(decoded.bookmarks, {'downloads': 'D:/Downloads'});
    expect(decoded.createHistory, ['https://example.com/file.zip']);
    expect(decoded.runAsMenubarApp, isTrue);
    expect(decoded.analyticsEnabled, isFalse);
    expect(decoded.analyticsClientId, 'client-id');
  });
}
