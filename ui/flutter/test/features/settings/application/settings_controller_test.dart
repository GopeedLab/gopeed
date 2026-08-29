import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/core/capabilities/app_capabilities.dart';
import 'package:gopeed/core/capabilities/capability_rpc.dart';
import 'package:gopeed/core/capabilities/gopeed_capability.dart';
import 'package:gopeed/core/common/start_config.dart';
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
}

class _TestRuntimeController extends AppRuntimeController {
  @override
  Future<AppRuntimeState> build() async => AppRuntimeState(
    startConfig: StartConfig(),
    runningPort: 0,
    downloaderConfig: DownloaderConfig(downloadDir: 'C:/Stale runtime snapshot'),
  );
}
