import 'package:flutter/widgets.dart';

import '../../../../../core/utils/breakpoints.dart';
import '../../../domain/task_record.dart';
import '../task_context_menu.dart';
import 'task_card_desktop.dart';
import 'task_card_mobile.dart';

class TaskCard extends StatelessWidget {
  const TaskCard({
    super.key,
    required this.task,
    required this.selected,
    required this.batchMode,
    required this.selectedInBatch,
    required this.onPressed,
    required this.onToggleBatch,
    this.onPause,
    this.onResume,
    this.onDelete,
    this.onOpen,
    this.onReveal,
    this.contextActions,
    this.listeningForUpdate = false,
  });

  final TaskRecord task;
  final bool selected;
  final bool batchMode;
  final bool selectedInBatch;
  final VoidCallback onPressed;
  final VoidCallback onToggleBatch;
  final VoidCallback? onPause;
  final VoidCallback? onResume;
  final VoidCallback? onDelete;
  final VoidCallback? onOpen;
  final VoidCallback? onReveal;
  final TaskContextActions? contextActions;
  final bool listeningForUpdate;

  @override
  Widget build(BuildContext context) {
    final isMobile = MediaQuery.sizeOf(context).width < Breakpoints.mobile;

    if (isMobile) {
      final card = TaskCardMobile(
        task: task,
        selected: selected,
        batchMode: batchMode,
        selectedInBatch: selectedInBatch,
        onPressed: onPressed,
        onToggleBatch: onToggleBatch,
        onPause: onPause,
        onResume: onResume,
        onDelete: onDelete,
        onOpen: onOpen,
        onReveal: onReveal,
        listeningForUpdate: listeningForUpdate,
      );
      final actions = contextActions;
      return actions == null ? card : TaskContextMenu(task: task, actions: actions, child: card);
    }

    final card = TaskCardDesktop(
      task: task,
      selected: selected,
      batchMode: batchMode,
      selectedInBatch: selectedInBatch,
      onPressed: onPressed,
      onToggleBatch: onToggleBatch,
      onPause: onPause,
      onResume: onResume,
      onDelete: onDelete,
      onOpen: onOpen,
      onReveal: onReveal,
      listeningForUpdate: listeningForUpdate,
    );
    final actions = contextActions;
    return actions == null ? card : TaskContextMenu(task: task, actions: actions, child: card);
  }
}
