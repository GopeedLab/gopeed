import 'dart:async';

import 'package:app_links_platform_interface/app_links_platform_interface.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/app/application/app_deep_link_controller.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:share_handler/share_handler.dart';

void main() {
  test('keeps the established path-style Gopeed deep-link routes', () {
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed:///create')), '/create');
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed:///extension')), '/extension');
  });

  test('rejects host-style Gopeed deep-link routes', () {
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed://create')), isEmpty);
    expect(gopeedDeepLinkRoute(Uri.parse('gopeed://extension')), isEmpty);
  });

  test('uses a shared attachment path before an optional caption', () {
    final media = SharedMedia(
      content: 'https://example.com/caption',
      attachments: [SharedAttachment(path: '/tmp/example.torrent', type: SharedAttachmentType.file)],
    );

    expect(sharedMediaUri(media), Uri.file('/tmp/example.torrent'));
  });

  test('uses shared text when there is no attachment', () {
    final media = SharedMedia(content: '  magnet:?xt=urn:btih:example  ');

    expect(sharedMediaUri(media), Uri.parse('magnet:?xt=urn:btih:example'));
  });

  test('keeps listening for browser takeover links after runtime settings change', () async {
    final originalPlatform = AppLinksPlatform.instance;
    final appLinksPlatform = _FakeAppLinksPlatform();
    AppLinksPlatform.instance = appLinksPlatform;
    final container = ProviderContainer(
      overrides: [appRuntimeControllerProvider.overrideWith(_FakeRuntimeController.new)],
    );

    try {
      await container.read(appDeepLinkControllerProvider.future);
      expect(appLinksPlatform.hasListener, isTrue);

      container.read(appRuntimeControllerProvider.notifier).replaceDownloaderConfig(DownloaderConfig());
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);

      expect(appLinksPlatform.hasListener, isTrue);
    } finally {
      container.dispose();
      await Future<void>.delayed(Duration.zero);
      AppLinksPlatform.instance = originalPlatform;
      await appLinksPlatform.close();
    }
  });
}

class _FakeAppLinksPlatform extends AppLinksPlatform {
  final _links = StreamController<Uri>.broadcast();

  bool get hasListener => _links.hasListener;

  @override
  Future<Uri?> getInitialLink() async => null;

  @override
  Stream<Uri> get uriLinkStream => _links.stream;

  Future<void> close() => _links.close();
}

class _FakeRuntimeController extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async {
    final config = StartConfig()
      ..network = 'tcp'
      ..address = '127.0.0.1:9999'
      ..apiEnable = true
      ..apiToken = ''
      ..storage = 'bolt'
      ..storageDir = ''
      ..refreshInterval = 0;
    return AppRuntimeState(
      startConfig: config,
      apiServerState: const ApiServerState(
        enabled: true,
        mcpEnabled: false,
        running: true,
        network: 'tcp',
        address: '127.0.0.1:9999',
        runningPort: 9999,
        pendingApply: false,
        lastError: '',
      ),
      downloaderConfig: DownloaderConfig(),
    );
  }
}
