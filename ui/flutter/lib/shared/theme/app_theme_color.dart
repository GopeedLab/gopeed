import 'package:flutter/widgets.dart';

enum AppThemeColor {
  green(
    key: 'green',
    light: Color(0xFF79C476),
    dark: Color(0xFFA7F5A2),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF07310B),
  ),
  red(
    key: 'red',
    light: Color(0xFFE5484D),
    dark: Color(0xFFFF8A8E),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF3B0507),
  ),
  rose(
    key: 'rose',
    light: Color(0xFFE93D82),
    dark: Color(0xFFFF8AB7),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF3B071B),
  ),
  orange(
    key: 'orange',
    light: Color(0xFFF76B15),
    dark: Color(0xFFFFA45B),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF3A1600),
  ),
  cyan(
    key: 'cyan',
    light: Color(0xFF0CA678),
    dark: Color(0xFF5EE0B5),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF003829),
  ),
  blue(
    key: 'blue',
    light: Color(0xFF2563EB),
    dark: Color(0xFF73A7FF),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF061B3F),
  ),
  amber(
    key: 'amber',
    light: Color(0xFFD99A00),
    dark: Color(0xFFFFD166),
    lightForeground: Color(0xFF211800),
    darkForeground: Color(0xFF302300),
  ),
  purple(
    key: 'purple',
    light: Color(0xFF8B5CF6),
    dark: Color(0xFFB794F6),
    lightForeground: Color(0xFFFFFFFF),
    darkForeground: Color(0xFF261052),
  );

  const AppThemeColor({
    required this.key,
    required this.light,
    required this.dark,
    required this.lightForeground,
    required this.darkForeground,
  });

  final String key;
  final Color light;
  final Color dark;
  final Color lightForeground;
  final Color darkForeground;

  static AppThemeColor fromKey(String? key) {
    return values.where((color) => color.key == key).firstOrNull ?? AppThemeColor.green;
  }
}

class AppThemePreviewColors {
  const AppThemePreviewColors._();

  static const lightBackground = Color(0xFFF7F8F8);
  static const lightSide = Color(0xFFE5E8E7);
  static const lightLine = Color(0xFFB8BEBC);
  static const darkBackground = Color(0xFF111318);
  static const darkSide = Color(0xFF20242C);
  static const darkLine = Color(0xFF676D78);
  static const systemSide = Color(0x55878E8C);
  static const systemLine = Color(0xFF7B8280);
  static const transparent = Color(0x00000000);
}
