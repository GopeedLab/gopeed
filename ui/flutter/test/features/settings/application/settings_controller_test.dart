import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/features/settings/application/settings_controller.dart';

void main() {
  test('settings reload backend config after the page releases its provider', () async {
    var backendDirectory = 'D:/Initial';
    var getConfigCalls = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) {
        getConfigCalls++;
        return DownloaderConfig(downloadDir: backendDirectory);
      });
    final container = ProviderContainer(
      overrides: [
        appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
        appRuntimeControllerProvider.overrideWith(_TestRuntimeController.new),
      ],
    );
    addTearDown(container.dispose);

    final firstListener = container.listen(settingsControllerProvider, (_, _) {}, fireImmediately: true);
    expect((await container.read(settingsControllerProvider.future)).config.downloadDir, 'D:/Initial');
    firstListener.close();
    await container.pump();

    backendDirectory = 'E:/Created task';
    final secondListener = container.listen(settingsControllerProvider, (_, _) {}, fireImmediately: true);
    addTearDown(secondListener.close);
    expect((await container.read(settingsControllerProvider.future)).config.downloadDir, 'E:/Created task');
    expect(getConfigCalls, 2);
  });

  test('settings save preserves backend-owned client preferences', () async {
    final backend = DownloaderConfig()
      ..extra.windowState = WindowStateConfig(isMaximized: true, width: 1200, height: 800)
      ..extra.createHistory = ['https://example.com/existing.zip']
      ..extra.deviceId = 'existing-client';
    DownloaderConfig? saved;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig.fromJson(backend.toJson()))
      ..bind(GopeedMethods.putConfig, (config) {
        saved = DownloaderConfig.fromJson(config.toJson());
        return const RpcUnit();
      });
    final container = ProviderContainer(
      overrides: [
        appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
        appRuntimeControllerProvider.overrideWith(_TestRuntimeController.new),
      ],
    );
    addTearDown(container.dispose);

    final controller = container.read(settingsControllerProvider.notifier);
    await container.read(settingsControllerProvider.future);
    await controller.save(DownloaderConfig(downloadDir: 'E:/New'));

    expect(saved?.downloadDir, 'E:/New');
    expect(saved?.extra.windowState.isMaximized, isTrue);
    expect(saved?.extra.createHistory, ['https://example.com/existing.zip']);
    expect(saved?.extra.deviceId, 'existing-client');
  });

  test('protocol edits use a single full PUT and reload backend-owned tracker results', () async {
    final backend = DownloaderConfig();
    final puts = <DownloaderConfig>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig.fromJson(backend.toJson()))
      ..bind(GopeedMethods.putConfig, (config) {
        puts.add(config);
        // Simulate the backend retaining its read-only results on a full PUT.
        backend.protocolConfig.bt.listenPort = config.protocolConfig.bt.listenPort;
        return const RpcUnit();
      });
    final container = _settingsContainer(registry);
    addTearDown(container.dispose);
    final initial = await container.read(settingsControllerProvider.future);
    final draft = DownloaderConfig.fromJson(initial.config.toJson())..protocolConfig.bt.listenPort = 6881;
    backend.protocolConfig.bt.state.trackerSources = {
      'https://example.com/list': ['udp://fresh.example:80/announce'],
    };
    backend.protocolConfig.bt.state.lastTrackerUpdateTime = DateTime.utc(2026, 10, 19);
    await container.read(settingsControllerProvider.notifier).save(draft);
    expect(puts, hasLength(1));
    expect(puts.single.protocolConfig.bt.listenPort, 6881);
    final saved = container.read(settingsControllerProvider).requireValue;
    expect(saved.config.protocolConfig.bt.state.trackerSources['https://example.com/list'], [
      'udp://fresh.example:80/announce',
    ]);
    expect(saved.saving, isFalse);
  });

  test('rapid full config saves run in order and keep the latest user edits', () async {
    final backend = DownloaderConfig();
    final entered = Completer<void>();
    final release = Completer<void>();
    final puts = <DownloaderConfig>[];
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig.fromJson(backend.toJson()))
      ..bind(GopeedMethods.putConfig, (config) async {
        puts.add(DownloaderConfig.fromJson(config.toJson()));
        if (puts.length == 1) {
          entered.complete();
          await release.future;
        }
        backend.protocolConfig = config.protocolConfig;
        return const RpcUnit();
      });
    final container = _settingsContainer(registry);
    addTearDown(container.dispose);
    final initial = await container.read(settingsControllerProvider.future);
    final controller = container.read(settingsControllerProvider.notifier);
    final first = DownloaderConfig.fromJson(initial.config.toJson())..protocolConfig.bt.listenPort = 6881;
    final savingFirst = controller.save(first);
    await entered.future;
    final second = DownloaderConfig.fromJson(first.toJson())
      ..protocolConfig.bt.customTrackers = ['udp://custom.example:80/announce'];
    final savingSecond = controller.save(second);
    expect(puts.length, 1);
    release.complete();
    await Future.wait([savingFirst, savingSecond]);
    expect(puts, hasLength(2));
    expect(puts[0].protocolConfig.bt.listenPort, 6881);
    expect(puts[1].protocolConfig.bt.listenPort, 6881);
    expect(puts[1].protocolConfig.bt.customTrackers, ['udp://custom.example:80/announce']);
    expect(container.read(settingsControllerProvider).requireValue.config.protocolConfig.bt.customTrackers, [
      'udp://custom.example:80/announce',
    ]);
  });

  test('a failed save clears saving state and the same change can be retried', () async {
    final backend = DownloaderConfig();
    var attempts = 0;
    final registry = CapabilityRegistry(createAppCapabilityCodecs())
      ..bind(GopeedMethods.getConfig, (_) => DownloaderConfig.fromJson(backend.toJson()))
      ..bind(GopeedMethods.putConfig, (config) {
        if (++attempts == 1) throw StateError('offline');
        backend.protocolConfig = config.protocolConfig;
        return const RpcUnit();
      });
    final container = _settingsContainer(registry);
    addTearDown(container.dispose);
    final initial = await container.read(settingsControllerProvider.future);
    final controller = container.read(settingsControllerProvider.notifier);
    final draft = DownloaderConfig.fromJson(initial.config.toJson())..protocolConfig.bt.listenPort = 6881;
    await expectLater(controller.save(draft), throwsStateError);
    expect(container.read(settingsControllerProvider).requireValue.saving, isFalse);
    await controller.save(draft);
    expect(attempts, 2);
    expect(backend.protocolConfig.bt.listenPort, 6881);
  });
}

ProviderContainer _settingsContainer(CapabilityRegistry registry) {
  final container = ProviderContainer(
    overrides: [
      appCapabilitiesProvider.overrideWithValue(AppCapabilities(LocalCapabilityInvoker(registry))),
      appRuntimeControllerProvider.overrideWith(_TestRuntimeController.new),
    ],
  );
  container.listen(settingsControllerProvider, (_, _) {});
  return container;
}

class _TestRuntimeController extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig(),
    apiServerState: const ApiServerState(
      enabled: false,
      mcpEnabled: false,
      running: false,
      network: '',
      address: '',
      runningPort: 0,
      pendingApply: false,
      lastError: '',
    ),
    downloaderConfig: DownloaderConfig(downloadDir: 'C:/Stale runtime snapshot'),
  );
}
