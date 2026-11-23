import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;
import 'package:flutter/material.dart' show Colors;
import 'package:flutter/services.dart' show MethodChannel;
import 'package:flutter/widgets.dart';
import 'package:window_manager/window_manager.dart';

import '../../api/model/downloader_config.dart';
import '../../l10n/l10n.dart';
import '../../shared/theme/app_design_tokens.dart';
import 'app_window_chrome.dart';
import 'app_window_payload.dart';

class AppWindowBootstrap {
  static Future<void> setLinuxProgressWindowSize(Size size) => const MethodChannel(
    'gopeed/progress_window',
  ).invokeMethod<void>('setSize', {'width': size.width.round(), 'height': size.height.ceil()});

  static Future<void> setupMainWindow({bool hidden = false}) async {
    if (!AppWindowChrome.isDesktopWindow) {
      return;
    }

    await windowManager.ensureInitialized();

    final windowOptions = WindowOptions(
      title: 'Gopeed',
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
      if (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.macOS) {
        await windowManager.setHasShadow(true);
      }
      if (!hidden) {
        await windowManager.show();
        await windowManager.focus();
      }
    });
  }

  static Future<void> restoreMainWindowState(WindowStateConfig windowState, {required bool hidden}) async {
    final width = windowState.width;
    final height = windowState.height;
    if (width != null && height != null && width > 0 && height > 0) {
      await windowManager.setSize(Size(width, height));
      await windowManager.center();
    }
    // On Windows, maximizing a hidden window also shows it.
    if (!hidden && windowState.isMaximized) {
      await windowManager.maximize();
    }
  }

  static Future<void> setupSubWindow(AppWindowType type, {String localeConfig = ''}) async {
    if (!AppWindowChrome.isDesktopWindow) {
      return;
    }

    await windowManager.ensureInitialized();
    final title = subWindowTitle(type, appLocalizationsFor(localeConfig));

    if (type == AppWindowType.downloadProgress) {
      if (defaultTargetPlatform == TargetPlatform.linux) {
        await setLinuxProgressWindowSize(AppDesignTokens.downloadProgressWindowSize);
      }
      final progressOptions = WindowOptions(
        size: AppDesignTokens.downloadProgressWindowSize,
        center: true,
        skipTaskbar: false,
        backgroundColor: Colors.transparent,
        title: title,
        titleBarStyle: TitleBarStyle.hidden,
      );
      await windowManager.waitUntilReadyToShow(progressOptions, () async {
        if (AppWindowChrome.usesCustomChrome) {
          await windowManager.setAsFrameless();
        }
        await windowManager.setResizable(false);
        await windowManager.setMaximizable(false);
        if (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.macOS) {
          await windowManager.setHasShadow(true);
        }
        await _showSubWindowInForeground();
      });
      return;
    }

    final options = switch (type) {
      AppWindowType.createTask => WindowOptions(
        size: AppDesignTokens.childWindowSize,
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
      if (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.macOS) {
        await windowManager.setHasShadow(true);
      }
      await _showSubWindowInForeground();
    });
  }

  static String subWindowTitle(AppWindowType type, AppLocalizations l10n) {
    return switch (type) {
      AppWindowType.createTask => 'Gopeed - ${l10n.create}',
      AppWindowType.downloadProgress => l10n.downloadProgressTitle,
      _ => 'Gopeed',
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
