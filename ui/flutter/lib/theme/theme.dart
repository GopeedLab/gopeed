import 'package:flutter/material.dart';

class GopeedTheme {
  // Deep teal primary — matches the screenshot's dark, vibrant palette
  static const _primaryValue = 0xFF00BCD4; // cyan/teal
  static const _primarySwatch = MaterialColor(_primaryValue, <int, Color>{
    50: Color(0xFFE0F7FA),
    100: Color(0xFFB2EBF2),
    200: Color(0xFF80DEEA),
    300: Color(0xFF4DD0E1),
    400: Color(0xFF26C6DA),
    500: Color(_primaryValue),
    600: Color(0xFF00ACC1),
    700: Color(0xFF0097A7),
    800: Color(0xFF00838F),
    900: Color(0xFF006064),
  });

  static const _accentValue = 0xFF18FFFF;
  static const _accentSwatch = MaterialColor(_accentValue, <int, Color>{
    100: Color(0xFFE4FFFE),
    200: Color(_accentValue),
    400: Color(0xFF00E5FF),
    700: Color(0xFF00B8D4),
  });

  // Light theme — minimal use case, keep consistent palette
  static final _light = ThemeData(
      useMaterial3: false,
      brightness: Brightness.light,
      primarySwatch: _primarySwatch);
  static final light = _light.copyWith(
      colorScheme: _light.colorScheme.copyWith(secondary: _accentSwatch));

  // Dark theme — pure black background, teal accents
  static final _dark = ThemeData(
      useMaterial3: false,
      brightness: Brightness.dark,
      primarySwatch: _primarySwatch,
      scaffoldBackgroundColor: const Color(0xFF000000),
      cardColor: const Color(0xFF111111),
      canvasColor: const Color(0xFF0A0A0A));
  static final dark = _dark.copyWith(
      colorScheme: _dark.colorScheme.copyWith(
        secondary: _accentSwatch,
        surface: const Color(0xFF111111),
        background: const Color(0xFF000000),
      ));
}

/// Category accent colors — used for task category badges and card borders,
/// matching the colorful card aesthetic in the design reference.
class DebridColors {
  static const audiobook = Color(0xFF4CAF50);  // green
  static const music     = Color(0xFF00BCD4);  // teal
  static const video     = Color(0xFF9C27B0);  // purple
  static const game      = Color(0xFFF44336);  // red
  static const software  = Color(0xFFFF9800);  // orange
  static const ebook     = Color(0xFFFFEB3B);  // yellow
  static const archive   = Color(0xFF2196F3);  // blue
  static const other     = Color(0xFF607D8B);  // blue-grey
}
