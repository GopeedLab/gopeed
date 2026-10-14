import 'package:flutter/widgets.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/widgets/app_empty_state.dart';
import 'task_empty_create_hint.dart';

class TaskEmptyState extends StatelessWidget {
  const TaskEmptyState({super.key, this.message, this.onCreateTask});

  final String? message;
  final VoidCallback? onCreateTask;

  @override
  Widget build(BuildContext context) {
    return AppEmptyState(
      message: message ?? context.l10n.emptyTaskList,
      illustration: AppEmptyIllustrationType.inbox,
      illustrationKey: const ValueKey('task-empty-illustration'),
      action: onCreateTask == null ? null : TaskEmptyCreateHint(onCreateTask: onCreateTask!),
    );
  }
}
