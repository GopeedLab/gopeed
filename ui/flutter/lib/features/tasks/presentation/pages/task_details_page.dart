import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../core/utils/file_explorer.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../l10n/l10n.dart';
import '../../application/tasks_controller.dart';
import '../../domain/task_record.dart';
import '../widgets/task_drawer.dart';

class TaskDetailsPage extends ConsumerWidget {
  const TaskDetailsPage({super.key, required this.taskId, this.initialTask});

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
                child: Row(
                  children: [
                    SizedBox(
                      width: 44,
                      child: shad.GhostButton(
                        density: shad.ButtonDensity.icon,
                        onPressed: () => context.canPop() ? context.pop() : context.go('/'),
                        child: Icon(Icons.arrow_back, size: 20, color: palette.textPrimary),
                      ),
                    ),
                    Expanded(
                      child: Text(
                        context.l10n.taskDetails,
                        textAlign: TextAlign.center,
                        style: TextStyle(color: palette.textPrimary, fontSize: 17, fontWeight: FontWeight.w700),
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
                  : TaskDetailsView(
                      key: ValueKey(task.id),
                      task: task,
                      mobile: true,
                      onOpenStorage: () => FileExplorer.reveal(task!.storagePath),
                      onUpdateUrl: (url) => ref
                          .read(tasksControllerProvider.notifier)
                          .updateUrl(task!.id, url, headers: task.requestHeaders),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
