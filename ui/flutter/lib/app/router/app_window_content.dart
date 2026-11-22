import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:flutter/widgets.dart';

import '../../api/model/create_task.dart';
import '../../core/window/app_window_payload.dart';
import '../../features/tasks/presentation/pages/create_task_window_page.dart';

/// Shared destination registry for desktop child engines and in-app windows.
abstract final class AppWindowContent {
  static String location(AppWindowType type) => switch (type) {
    AppWindowType.createTask => '/create',
    AppWindowType.main => throw ArgumentError.value(type, 'type', 'Expected a child window'),
  };

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
    AppWindowType.main => throw ArgumentError.value(payload.type, 'payload.type', 'Expected a child window'),
  };
}
