import 'dart:async';

import 'package:shared_preferences/shared_preferences.dart';

class WindowState {
  late bool isMaximized;
  late double width;
  late double height;

  WindowState(this.isMaximized, this.width, this.height);
}

const String _windowStateIsMaximized = "window.isMaximized";
const String _windowStateWidth = "window.width";
const String _windowStateHeight = "window.height";

/// Save window state to storage.
void saveState({bool? isMaximized, double? width, double? height}) async {
  final prefs = await SharedPreferences.getInstance();
  if (isMaximized != null) {
    await prefs.setBool(_windowStateIsMaximized, isMaximized);
  }
  if (width != null) {
    await prefs.setDouble(_windowStateWidth, width);
  }
  if (height != null) {
    await prefs.setDouble(_windowStateHeight, height);
  }
}

/// Load window state from storage.
Future<WindowState> loadState() async {
  final prefs = await SharedPreferences.getInstance();
  return WindowState(
    prefs.getBool(_windowStateIsMaximized) ?? false,
    prefs.getDouble(_windowStateWidth) ?? 800,
    prefs.getDouble(_windowStateHeight) ?? 600,
  );
}
