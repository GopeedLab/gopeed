import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/model/downloader_config.dart';
import '../../../app/application/app_runtime_controller.dart';
import '../../../core/capabilities/app_capabilities.dart';

final settingsControllerProvider = AsyncNotifierProvider<SettingsController, SettingsState>(SettingsController.new);

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
    final runtime = await ref.read(appRuntimeControllerProvider.future);
    return SettingsState(config: DownloaderConfig.fromJson(runtime.downloaderConfig.toJson()));
  }

  Future<void> reload() async {
    state = const AsyncValue.loading();
    state = AsyncValue.data(SettingsState(config: await ref.read(gopeedServiceProvider).getConfig()));
  }

  Future<void> save(DownloaderConfig config) async {
    state = AsyncValue.data(SettingsState(config: config, saving: true));
    await ref.read(gopeedServiceProvider).putConfig(config);
    ref.read(appRuntimeControllerProvider.notifier).replaceDownloaderConfig(config);
    state = AsyncValue.data(SettingsState(config: config));
  }

  Future<void> testWebhook(String url) async {
    await ref.read(gopeedServiceProvider).testWebhook(url);
  }
}
