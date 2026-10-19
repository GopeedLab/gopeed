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
  Future<void> _saveTail = Future<void>.value();
  int _saveRevision = 0;

  @override
  Future<SettingsState> build() async {
    await ref.read(appRuntimeControllerProvider.future);
    final config = await ref.read(gopeedServiceProvider).getConfig();
    return SettingsState(config: config);
  }

  Future<void> reload({bool showLoading = true}) async {
    if (showLoading) state = const AsyncValue.loading();
    final config = await ref.read(gopeedServiceProvider).getConfig();
    state = AsyncValue.data(SettingsState(config: config));
  }

  Future<void> save(DownloaderConfig config) {
    final snapshot = DownloaderConfig.fromJson(config.toJson());
    final revision = ++_saveRevision;
    state = AsyncValue.data(SettingsState(config: config, saving: true));
    final keepAlive = ref.keepAlive();
    final result = _saveTail.then((_) => _save(snapshot, revision)).whenComplete(keepAlive.close);
    // A failed save must not prevent subsequent edits or retries from running.
    _saveTail = result.then((_) {}, onError: (Object _, StackTrace _) {});
    return result;
  }

  Future<void> _save(DownloaderConfig config, int revision) async {
    try {
      final service = ref.read(gopeedServiceProvider);
      final latest = await service.getConfig();
      config.extra
        ..windowState = latest.extra.windowState
        ..bookmarks = latest.extra.bookmarks
        ..createHistory = latest.extra.createHistory
        ..runAsMenubarApp = latest.extra.runAsMenubarApp
        ..analyticsClientId = latest.extra.analyticsClientId;
      // The backend preserves protocol-owned results and validates the whole PUT.
      await service.putConfig(config);
      final saved = await service.getConfig();
      if (ref.mounted) {
        ref.read(appRuntimeControllerProvider.notifier).replaceDownloaderConfig(saved);
        if (revision == _saveRevision) state = AsyncValue.data(SettingsState(config: saved));
      }
    } finally {
      if (ref.mounted && revision == _saveRevision && state.value?.saving == true) {
        state = AsyncValue.data(state.requireValue.copyWith(saving: false));
      }
    }
  }

  Future<void> testWebhook(String url) async {
    await ref.read(gopeedServiceProvider).testWebhook(url);
  }
}
