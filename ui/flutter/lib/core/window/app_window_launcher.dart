import 'dart:io';

import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:flutter/foundation.dart';
import 'package:window_manager/window_manager.dart';

import '../../util/log_util.dart';
import '../capabilities/app_capabilities.dart';
import 'app_window_payload.dart';
import 'window_capability_transport.dart';

class AppWindowLauncher {
  static Future<bool> open(AppWindowPayload payload) async {
    if (payload.type == AppWindowType.main) {
      throw ArgumentError.value(payload.type, 'payload.type', 'Expected a child window');
    }
    if (kIsWeb || !(Platform.isWindows || Platform.isMacOS || Platform.isLinux)) {
      return false;
    }

    try {
      await AppWindowCapabilityHost.instance.start(LocalAppCapabilities.instance.registry);
      // Every request deliberately creates a new child window. Create-task
      // windows are independent and are never reused.
      await WindowController.create(WindowConfiguration(arguments: payload.toRaw(), hiddenAtLaunch: true));
      // Keep the native window hidden until the child engine has applied its
      // final size, centered position, chrome, and foreground behavior in
      // AppWindowBootstrap.setupSubWindow.
      return true;
    } catch (error, stackTrace) {
      try {
        logger.w('child window failed', error, stackTrace);
      } catch (_) {
        debugPrint('child window failed: $error');
      }
      // A tray/deep-link request can arrive while the main window is hidden.
      // Bring the in-app fallback into view if the child engine cannot start.
      try {
        await windowManager.show();
        await windowManager.focus();
      } catch (_) {}
      return false;
    }
  }
}
