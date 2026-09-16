import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/downloader_config.dart';
import 'package:gopeed/app/application/app_platform_controller.dart';
import 'package:gopeed/app/application/app_runtime_controller.dart';
import 'package:gopeed/core/common/api_server_state.dart';
import 'package:gopeed/core/common/start_config.dart';
import 'package:gopeed/util/updater.dart';

void main() {
  test('runtime updates do not rebuild the platform controller', () async {
    final container = ProviderContainer(
      overrides: [
        appRuntimeControllerProvider.overrideWith(_TestRuntimeController.new),
        appPlatformControllerProvider.overrideWith(_TestPlatformController.new),
      ],
    );
    addTearDown(container.dispose);
    final subscription = container.listen(appPlatformControllerProvider, (_, _) {}, fireImmediately: true);
    addTearDown(subscription.close);

    await container.read(appPlatformControllerProvider.future);
    final platformController = container.read(appPlatformControllerProvider.notifier) as _TestPlatformController;
    final runtimeController = container.read(appRuntimeControllerProvider.notifier) as _TestRuntimeController;

    expect(platformController.buildCount, 1);

    runtimeController.completeRuntime();
    await container.read(appRuntimeControllerProvider.future);
    await Future<void>.delayed(Duration.zero);
    expect(platformController.updateCheckCount, 1);

    final config = DownloaderConfig.fromJson(runtimeController.state.requireValue.downloaderConfig.toJson());
    config.extra.locale = 'zh';
    runtimeController.replaceDownloaderConfig(config);
    await container.pump();

    expect(platformController.buildCount, 1);
  });
}

class _TestPlatformController extends AppPlatformController {
  int buildCount = 0;
  int updateCheckCount = 0;

  @override
  bool get isDesktop => false;

  @override
  Future<AppPlatformState> build() {
    buildCount++;
    return super.build();
  }

  @override
  Future<VersionInfo?> checkForUpdate() async {
    updateCheckCount++;
    return null;
  }
}

class _TestRuntimeController extends AppRuntimeController {
  final _pendingRuntime = Completer<AppRuntimeState>();

  @override
  Future<AppRuntimeState> build() => _pendingRuntime.future;

  void completeRuntime() {
    _pendingRuntime.complete(
      AppRuntimeState(
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
        downloaderConfig: DownloaderConfig(),
      ),
    );
  }
}
