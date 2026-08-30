import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;
import 'package:flutter/material.dart' show Colors;
import 'package:flutter/widgets.dart';
import 'package:window_manager/window_manager.dart';

import '../../l10n/l10n.dart';
import 'app_window_chrome.dart';
import 'app_window_payload.dart';

class AppWindowBootstrap {
  static Future<void> setupMainWindow({bool hidden = false}) async {
    if (!AppWindowChrome.isDesktopWindow) {
      return;
    }

    await windowManager.ensureInitialized();

    final windowOptions = WindowOptions(
      size: const Size(1024, 768),
      center: true,
      skipTaskbar: false,
      backgroundColor: Colors.transparent,
      titleBarStyle: TitleBarStyle.hidden,
    );

    windowManager.waitUntilReadyToShow(windowOptions, () async {
      if (AppWindowChrome.usesCustomChrome) {
        await windowManager.setAsFrameless();
      }
      await windowManager.setHasShadow(true);
      if (!hidden) {
        await windowManager.show();
        await windowManager.focus();
      }
    });
  }

  static Future<void> setupSubWindow(AppWindowType type, {String localeConfig = ''}) async {
    if (!AppWindowChrome.isDesktopWindow) {
      return;
    }

    await windowManager.ensureInitialized();
    final title = subWindowTitle(type, appLocalizationsFor(localeConfig));

    final options = switch (type) {
      AppWindowType.createTask => WindowOptions(
        size: Size(820, 620),
        center: true,
        skipTaskbar: false,
        backgroundColor: Colors.transparent,
        title: title,
        titleBarStyle: TitleBarStyle.hidden,
      ),
      _ => WindowOptions(
        size: Size(900, 640),
        center: true,
        skipTaskbar: false,
        backgroundColor: Colors.transparent,
        title: title,
        titleBarStyle: TitleBarStyle.hidden,
      ),
    };

    windowManager.waitUntilReadyToShow(options, () async {
      if (AppWindowChrome.usesCustomChrome) {
        await windowManager.setAsFrameless();
      }
      await windowManager.setTitle(title);
      await windowManager.setHasShadow(true);
      await _showSubWindowInForeground();
    });
  }

  static String subWindowTitle(AppWindowType type, AppLocalizations l10n) {
    return switch (type) {
      AppWindowType.createTask => '${l10n.create} - Gopeed',
      _ => l10n.appTitle,
    };
  }

  static Future<void> _showSubWindowInForeground() async {
    if (defaultTargetPlatform != TargetPlatform.windows) {
      await windowManager.show();
      await windowManager.focus();
      return;
    }

    // SetForegroundWindow may be rejected after the browser regains focus
    // while the child Flutter engine is starting. A brief topmost pulse keeps
    // the user-initiated create window visible without making it permanently
    // always-on-top.
    var pulsing = false;
    try {
      await windowManager.setAlwaysOnTop(true);
      pulsing = true;
      await windowManager.show();
      await windowManager.focus();
      await Future<void>.delayed(const Duration(milliseconds: 180));
    } finally {
      if (pulsing) {
        await windowManager.setAlwaysOnTop(false);
      }
    }
    await windowManager.focus();
  }
}
