import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/model/downloader_config.dart';
import '../../shared/theme/app_theme_color.dart';

enum AppThemeMode {
  system('system'),
  light('light'),
  dark('dark');

  const AppThemeMode(this.key);

  final String key;

  static AppThemeMode fromKey(String? key) {
    return values.where((mode) => mode.key == key).firstOrNull ?? AppThemeMode.system;
  }
}

class AppAppearanceState {
  const AppAppearanceState({this.themeMode = AppThemeMode.system, this.themeColor = AppThemeColor.green});

  final AppThemeMode themeMode;
  final AppThemeColor themeColor;

  bool resolveIsLight(Brightness systemBrightness) {
    return switch (themeMode) {
      AppThemeMode.system => systemBrightness == Brightness.light,
      AppThemeMode.light => true,
      AppThemeMode.dark => false,
    };
  }

  AppAppearanceState copyWith({AppThemeMode? themeMode, AppThemeColor? themeColor}) {
    return AppAppearanceState(themeMode: themeMode ?? this.themeMode, themeColor: themeColor ?? this.themeColor);
  }
}

final appAppearanceControllerProvider = NotifierProvider<AppAppearanceController, AppAppearanceState>(
  AppAppearanceController.new,
);

class AppAppearanceController extends Notifier<AppAppearanceState> {
  bool _initialized = false;

  @override
  AppAppearanceState build() => const AppAppearanceState();

  void initialize(ExtraConfig config) {
    if (_initialized) return;
    _initialized = true;
    state = AppAppearanceState(
      themeMode: AppThemeMode.fromKey(config.themeMode),
      themeColor: AppThemeColor.fromKey(config.themeColor),
    );
  }

  void setThemeMode(AppThemeMode mode) {
    _initialized = true;
    state = state.copyWith(themeMode: mode);
  }

  void setThemeColor(AppThemeColor color) {
    _initialized = true;
    state = state.copyWith(themeColor: color);
  }
}
