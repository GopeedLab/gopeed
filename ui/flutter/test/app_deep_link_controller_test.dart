import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:gopeed/features/extensions/application/pending_extension_install.dart';

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
  TestWidgetsFlutterBinding.ensureInitialized();

  test('receives initial and subsequent scheme extension URLs with UTF-8 params', () async {
    final originalPlatform = AppLinksPlatform.instance;
    final initialUrl = 'https://github.com/author/扩展';
    Uri link(String url) => Uri.parse('gopeed:///extension').replace(
      queryParameters: {
        'params': base64Encode(utf8.encode(jsonEncode({'url': url}))),
      },
    );
    final platform = _FakeAppLinksPlatform(initialLink: link(initialUrl));
    AppLinksPlatform.instance = platform;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
      const MethodChannel('window_manager'),
      (_) async => false,
    );
    final container = ProviderContainer(
      overrides: [appRuntimeControllerProvider.overrideWith(_FakeRuntimeController.new)],
    );
    try {
      await container.read(appDeepLinkControllerProvider.future);
      expect(container.read(pendingExtensionInstallProvider)?.url, initialUrl);
      container.read(pendingExtensionInstallProvider.notifier).clear();
      platform.emit(link('https://github.com/author/second'));
      await Future<void>.delayed(Duration.zero);
      expect(container.read(pendingExtensionInstallProvider)?.url, 'https://github.com/author/second');
    } finally {
      container.dispose();
      await Future<void>.delayed(Duration.zero);
      AppLinksPlatform.instance = originalPlatform;
      await platform.close();
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
        const MethodChannel('window_manager'),
        null,
      );
    }
  });

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
  _FakeAppLinksPlatform({this.initialLink});

  final Uri? initialLink;
  void emit(Uri uri) => _links.add(uri);
  final _links = StreamController<Uri>.broadcast();

  bool get hasListener => _links.hasListener;

  @override
  Future<Uri?> getInitialLink() async => initialLink;

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
