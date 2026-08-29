import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' as material;
import 'package:flutter/painting.dart' show TextStyle;
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import 'app_palette.dart';
import 'app_theme_color.dart';

class AppTheme {
  const AppTheme._();

  // Flutter's automatic CJK fallback can select faces with mismatched weights
  // on Windows. Keep Geist for supported glyphs, but make the native Windows
  // UI font the deterministic fallback for Chinese text.
  static const _windowsFontFamilyFallback = <String>['Microsoft YaHei UI'];

  static bool get _usesWindowsFontFallback => !kIsWeb && defaultTargetPlatform == TargetPlatform.windows;

  static shad.Typography _typography() {
    const typography = shad.Typography.geist();
    if (!_usesWindowsFontFallback) return typography;

    return typography.copyWith(
      // Do not pass `package` here: Flutter prefixes every fallback family
      // with the package name when it is set, including system font names.
      sans: () => const TextStyle(
        fontFamily: 'packages/shadcn_flutter/GeistSans',
        fontFamilyFallback: _windowsFontFamilyFallback,
      ),
    );
  }

  static shad.ThemeData light([AppThemeColor themeColor = AppThemeColor.green]) {
    return _buildTheme(palette: AppPalette.lightFor(themeColor), brightness: Brightness.light);
  }

  static shad.ThemeData dark([AppThemeColor themeColor = AppThemeColor.green]) {
    return _buildTheme(palette: AppPalette.darkFor(themeColor), brightness: Brightness.dark);
  }

  static shad.ThemeData _buildTheme({required AppPalette palette, required Brightness brightness}) {
    return shad.ThemeData(
      colorScheme: shad.ColorScheme.fromColors(
        brightness: brightness,
        colors: {
          'background': palette.bg,
          'foreground': palette.textPrimary,
          'card': palette.cardBg,
          'cardForeground': palette.textPrimary,
          'popover': palette.sideBg,
          'popoverForeground': palette.textPrimary,
          'primary': palette.brand,
          'primaryForeground': palette.brandForeground,
          'secondary': palette.surfaceSoft,
          'secondaryForeground': palette.textPrimary,
          'muted': palette.surfaceSoft,
          'mutedForeground': palette.textSecondary,
          'accent': palette.inputBg,
          'accentForeground': palette.textPrimary,
          'destructive': palette.error,
          'destructiveForeground': palette.bg,
          'border': palette.border,
          'input': palette.border,
          'ring': palette.brand,
          'chart1': palette.brand,
          'chart2': palette.textSecondary,
          'chart3': palette.textMuted,
          'chart4': palette.surfaceSoft,
          'chart5': palette.cardHoverBg,
        },
      ),
      typography: _typography(),
      radius: 0.5,
      scaling: 1,
      surfaceBlur: 0,
      surfaceOpacity: 1,
      platform: defaultTargetPlatform,
    );
  }

  static material.ThemeData materialLight([AppThemeColor themeColor = AppThemeColor.green]) {
    return _buildMaterialTheme(palette: AppPalette.lightFor(themeColor), brightness: Brightness.light);
  }

  static material.ThemeData materialDark([AppThemeColor themeColor = AppThemeColor.green]) {
    return _buildMaterialTheme(palette: AppPalette.darkFor(themeColor), brightness: Brightness.dark);
  }

  static material.ThemeData _buildMaterialTheme({required AppPalette palette, required Brightness brightness}) {
    final theme = material.ThemeData.from(
      colorScheme: material.ColorScheme.fromSeed(
        seedColor: palette.brand,
        brightness: brightness,
        surface: palette.bg,
        primary: palette.brand,
      ),
      useMaterial3: true,
    );
    return theme.copyWith(
      extensions: <material.ThemeExtension<dynamic>>[palette],
      textTheme: _usesWindowsFontFallback
          ? theme.textTheme.apply(fontFamilyFallback: _windowsFontFamilyFallback)
          : theme.textTheme,
      primaryTextTheme: _usesWindowsFontFallback
          ? theme.primaryTextTheme.apply(fontFamilyFallback: _windowsFontFamilyFallback)
          : theme.primaryTextTheme,
    );
  }
}
