import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_tooltip.dart';
import '../../application/tasks_controller.dart';
import '../../domain/task_record.dart';
import '../widgets/task_file_manager.dart';

class TaskFilesPage extends ConsumerWidget {
  const TaskFilesPage({super.key, required this.taskId, this.initialTask});

  final String taskId;
  final TaskRecord? initialTask;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tasksAsync = ref.watch(tasksControllerProvider);
    TaskRecord? task = initialTask;
    for (final item in tasksAsync.value?.tasks ?? const <TaskRecord>[]) {
      if (item.id == taskId) {
        task = item;
        break;
      }
    }
    final palette = AppPalette.of(context);

    return shad.Scaffold(
      backgroundColor: palette.bg,
      child: ColoredBox(
        color: palette.bg,
        child: Column(
          children: [
            SafeArea(
              bottom: false,
              child: Container(
                height: 52,
                padding: const EdgeInsets.symmetric(horizontal: 8),
                decoration: BoxDecoration(
                  border: Border(bottom: BorderSide(color: palette.border)),
                ),
                child: Row(
                  children: [
                    SizedBox(
                      width: 44,
                      child: AppTooltip(
                        message: context.l10n.close,
                        child: shad.IconButton.ghost(
                          onPressed: () => context.canPop() ? context.pop() : context.go('/tasks/$taskId'),
                          icon: Icon(Icons.arrow_back, size: 20, color: palette.textPrimary),
                        ),
                      ),
                    ),
                    Expanded(
                      child: Text(
                        task?.name ?? context.l10n.taskDetailsFilesTab,
                        textAlign: TextAlign.center,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: palette.textPrimary, fontSize: 16, fontWeight: FontWeight.w700),
                      ),
                    ),
                    const SizedBox(width: 44),
                  ],
                ),
              ),
            ),
            Expanded(
              child: task == null
                  ? Center(
                      child: tasksAsync.isLoading
                          ? const shad.CircularProgressIndicator()
                          : Text(context.l10n.taskNotFound, style: TextStyle(color: palette.textMuted, fontSize: 13)),
                    )
                  : TaskFileManagerView(task: task),
            ),
          ],
        ),
      ),
    );
  }
}
