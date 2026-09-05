import 'dart:io';

import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:flutter/foundation.dart';

import '../../api/model/create_task.dart';
import '../../util/log_util.dart';
import '../capabilities/app_capabilities.dart';
import 'app_window_payload.dart';
import 'window_capability_transport.dart';

class AppWindowLauncher {
  static Future<bool> openCreateTaskWindow({CreateTask? createTask}) async {
    if (kIsWeb || !(Platform.isWindows || Platform.isMacOS || Platform.isLinux)) {
      return false;
    }

    try {
      await AppWindowCapabilityHost.instance.start(LocalAppCapabilities.instance.registry);
      final payload = AppWindowPayload.createTask(createTask: createTask?.toJson());
      // Every request deliberately creates a new child window. Create-task
      // windows are independent and are never reused.
      await WindowController.create(WindowConfiguration(arguments: payload.toRaw(), hiddenAtLaunch: true));
      // Keep the native window hidden until the child engine has applied its
      // final size, centered position, chrome, and foreground behavior in
      // AppWindowBootstrap.setupSubWindow.
      return true;
    } catch (error, stackTrace) {
      try {
        logger.w('create task window failed', error, stackTrace);
      } catch (_) {
        debugPrint('create task window failed: $error');
      }
      return false;
    }
  }
}
