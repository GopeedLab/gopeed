import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

import '../../../app/application/app_runtime_controller.dart';
import '../../../app/router/app_router.dart';
import '../../../core/window/app_window_chrome.dart';
import 'tasks_controller.dart';

final downloadProgressWindowControllerProvider = NotifierProvider<DownloadProgressWindowController, void>(
  DownloadProgressWindowController.new,
);

/// Observes the main window's task list, including tasks created by external clients.
class DownloadProgressWindowController extends Notifier<void> {
  // Captures from this run can already be present in the first task snapshot.
  final String captureSession = const Uuid().v4();

  @override
  void build() {
    if (!AppWindowChrome.isDesktopWindow) return;
    final seen = <String>{};
    ref.listen(tasksControllerProvider, (_, next) {
      final tasks = next.value;
      if (tasks == null || next.isLoading || next.hasError || tasks.lastError != null) return;
      final newTasks = tasks.tasks
          .where((task) => task.browserCaptureSession == captureSession && !seen.contains(task.id))
          .toList();
      seen.addAll(tasks.allTasks.where((task) => task.browserCaptureSession == captureSession).map((task) => task.id));
      final enabled = ref.read(appRuntimeControllerProvider).value?.downloaderConfig.extra.showDownloadProgressWindow;
      if (enabled == true) {
        for (final task in newTasks) {
          unawaited(_open(task.id));
        }
      }
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
  Future<void> openWindow(String taskId) async {
    await AppRouter.openDownloadProgress(taskId);
  }
}
