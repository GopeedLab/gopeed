import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/downloader_config.dart';
import '../../../app/application/app_runtime_controller.dart';
import '../../../core/capabilities/app_capabilities.dart';

final settingsControllerProvider = AsyncNotifierProvider.autoDispose<SettingsController, SettingsState>(
  SettingsController.new,
);

class SettingsState {
  const SettingsState({required this.config, this.saving = false});

  final DownloaderConfig config;
  final bool saving;

  SettingsState copyWith({DownloaderConfig? config, bool? saving}) {
    return SettingsState(config: config ?? this.config, saving: saving ?? this.saving);
  }
}

class SettingsController extends AsyncNotifier<SettingsState> {
  @override
  Future<SettingsState> build() async {
    await ref.read(appRuntimeControllerProvider.future);
    return SettingsState(config: await ref.read(gopeedServiceProvider).getConfig());
  }

  Future<void> reload({bool showLoading = true}) async {
    if (showLoading) state = const AsyncValue.loading();
    state = AsyncValue.data(SettingsState(config: await ref.read(gopeedServiceProvider).getConfig()));
  }

  Future<void> save(DownloaderConfig config) async {
    state = AsyncValue.data(SettingsState(config: config, saving: true));
    final latest = await ref.read(gopeedServiceProvider).getConfig();
    config.extra
      ..windowState = latest.extra.windowState
      ..bookmarks = latest.extra.bookmarks
      ..createHistory = latest.extra.createHistory
      ..runAsMenubarApp = latest.extra.runAsMenubarApp
      ..analyticsClientId = latest.extra.analyticsClientId;
    await ref.read(gopeedServiceProvider).putConfig(config);
    ref.read(appRuntimeControllerProvider.notifier).replaceDownloaderConfig(config);
    state = AsyncValue.data(SettingsState(config: config));
  }

  Future<void> testWebhook(String url) async {
    await ref.read(gopeedServiceProvider).testWebhook(url);
  }
}
