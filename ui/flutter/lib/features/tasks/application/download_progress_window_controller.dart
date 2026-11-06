import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/application/app_runtime_controller.dart';
import '../../../app/router/app_router.dart';
import '../../../core/window/app_window_chrome.dart';
import '../../../core/window/app_window_payload.dart';
import 'tasks_controller.dart';

final downloadProgressWindowControllerProvider = NotifierProvider<DownloadProgressWindowController, void>(
  DownloadProgressWindowController.new,
);

/// Observes the main window's task list, including tasks created by external clients.
class DownloadProgressWindowController extends Notifier<void> {
  @override
  void build() {
    if (!AppWindowChrome.isDesktopWindow) return;
    final seen = <String>{};
    var initialized = false;
    ref.listen(tasksControllerProvider, (_, next) {
      final tasks = next.value;
      if (tasks == null || next.isLoading || next.hasError || tasks.lastError != null) return;
      final newTasks = tasks.tasks.where((task) => !seen.contains(task.id)).toList();
      seen.addAll(tasks.allTasks.map((task) => task.id));
      // Establish a baseline so restarting the app never opens windows for old tasks.
      final enabled = ref.read(appRuntimeControllerProvider).value?.downloaderConfig.extra.showDownloadProgressWindow;
      if (initialized && enabled == true) {
        for (final task in newTasks) {
          unawaited(_open(task.id));
        }
      }
      initialized = true;
    }, fireImmediately: true);
  }

  Future<void> _open(String taskId) async {
    try {
      await openWindow(taskId);
    } catch (error) {
      debugPrint('Unable to open download progress window: $error');
    }
  }

  @visibleForTesting
  Future<void> openWindow(String taskId) => AppRouter.openWindow(AppWindowPayload.downloadProgress(taskId));
}
