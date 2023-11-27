import 'package:flutter/material.dart';

class GopeedTheme {
  static const _gopeedreenPrimaryValue = 0xFF79C476;
  static const _gopeedreen =
      MaterialColor(_gopeedreenPrimaryValue, <int, Color>{
    50: Color(0xFFEFF8EF),
    100: Color(0xFFD7EDD6),
    200: Color(0xFFBCE2BB),
    300: Color(0xFFA1D69F),
    400: Color(0xFF8DCD8B),
    500: Color(_gopeedreenPrimaryValue),
    600: Color(0xFF71BE6E),
    700: Color(0xFF66B663),
    800: Color(0xFF5CAF59),
    900: Color(0xFF49A246),
  });

  static const _gopeedreenAccentValue = 0xFFC9FFC7;
  static const _gopeedreenAccent =
      MaterialColor(_gopeedreenAccentValue, <int, Color>{
    100: Color(0xFFFAFFFA),
    200: Color(_gopeedreenAccentValue),
    400: Color(0xFF97FF94),
    700: Color(0xFF7FFF7A),
  });

  static final _light = ThemeData(
      useMaterial3: false,
      brightness: Brightness.light,
      primarySwatch: _gopeedreen);
  static final light = _light.copyWith(
      colorScheme: _light.colorScheme.copyWith(secondary: _gopeedreenAccent));

  static final _dark = ThemeData(
      useMaterial3: false,
      brightness: Brightness.dark,
      primarySwatch: _gopeedreen);
  static final dark = _dark.copyWith(
      colorScheme: _dark.colorScheme.copyWith(secondary: _gopeedreenAccent));
}
