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
      ..extra.analyticsClientId = 'existing-client';
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
    expect(saved?.extra.analyticsClientId, 'existing-client');
  });
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
