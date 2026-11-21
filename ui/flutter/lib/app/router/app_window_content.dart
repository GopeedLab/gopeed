import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:flutter/widgets.dart';

import '../../api/model/create_task.dart';
import '../../core/window/app_window_payload.dart';
import '../../features/tasks/presentation/pages/create_task_window_page.dart';
import '../../features/tasks/presentation/pages/download_progress_window_page.dart';

/// Shared content builders for native child windows and in-app task creation.
abstract final class AppWindowContent {
  static Widget build(
    AppWindowPayload payload, {
    WindowController? windowController,
    ValueChanged<bool>? onClose,
    bool modal = false,
  }) => switch (payload.type) {
    AppWindowType.createTask => CreateTaskWindowPage(
      windowController: windowController,
      initialTask: payload.createTask == null ? null : CreateTask.fromJson(payload.createTask!),
      onClose: onClose,
      modal: modal,
    ),
    AppWindowType.downloadProgress => DownloadProgressWindowPage(taskId: payload.taskId),
    AppWindowType.main => throw ArgumentError.value(payload.type, 'payload.type', 'Expected a child window'),
  };
}
