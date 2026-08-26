import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';
import 'package:window_manager/window_manager.dart';

import '../../database/database.dart';
import 'app_window_payload.dart';

class AppWindowBootstrap {
  static bool get _supportsDesktopWindows =>
      !kIsWeb &&
      (defaultTargetPlatform == TargetPlatform.windows ||
          defaultTargetPlatform == TargetPlatform.macOS ||
          defaultTargetPlatform == TargetPlatform.linux);

  static bool get _shouldRemoveNativeChrome =>
      !kIsWeb && (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.linux);

  static Future<void> setupMainWindow({bool hidden = false}) async {
    if (!_supportsDesktopWindows) {
      return;
    }

    await windowManager.ensureInitialized();

    final windowState = Database.instance.getWindowState();
    final windowOptions = WindowOptions(
      size: Size(windowState?.width ?? 1024, windowState?.height ?? 768),
      center: true,
      skipTaskbar: false,
      backgroundColor: Colors.transparent,
      titleBarStyle: TitleBarStyle.hidden,
    );

    windowManager.waitUntilReadyToShow(windowOptions, () async {
      if (_shouldRemoveNativeChrome) {
        await windowManager.setAsFrameless();
      }
      await windowManager.setHasShadow(true);
      if (!hidden) {
        await windowManager.show();
        await windowManager.focus();
      }
    });
  }

  static Future<void> setupSubWindow(AppWindowType type) async {
    if (!_supportsDesktopWindows) {
      return;
    }

    await windowManager.ensureInitialized();

    final options = switch (type) {
      AppWindowType.createTask => const WindowOptions(
        size: Size(820, 620),
        center: true,
        skipTaskbar: false,
        backgroundColor: Colors.transparent,
        titleBarStyle: TitleBarStyle.hidden,
      ),
      _ => const WindowOptions(
        size: Size(900, 640),
        center: true,
        skipTaskbar: false,
        backgroundColor: Colors.transparent,
        titleBarStyle: TitleBarStyle.hidden,
      ),
    };

    windowManager.waitUntilReadyToShow(options, () async {
      if (_shouldRemoveNativeChrome) {
        await windowManager.setAsFrameless();
      }
      await windowManager.setHasShadow(true);
      await windowManager.show();
      await windowManager.focus();
    });
  }
}
