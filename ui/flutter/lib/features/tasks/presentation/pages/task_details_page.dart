import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../core/utils/file_explorer.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../../../shared/widgets/detail/app_detail_surface.dart';
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

    return AppDetailPage(
      title: context.l10n.taskDetails,
      onBack: () => context.canPop() ? context.pop() : context.go('/'),
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
              onUpdateUrl: (url) async {
                try {
                  await ref
                      .read(tasksControllerProvider.notifier)
                      .updateUrl(task!.id, url, headers: task.requestHeaders);
                } catch (error) {
                  if (context.mounted) {
                    showAppToast(context, error.toString(), type: AppToastType.error);
                  }
                  rethrow;
                }
              },
            ),
    );
  }
}
