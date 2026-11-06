import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../core/utils/file_explorer.dart';
import '../../../../core/window/app_window_chrome.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../application/tasks_controller.dart';
import '../../domain/task_record.dart';
import '../widgets/download_progress_header.dart';
import '../widgets/download_progress_actions.dart';
import '../widgets/download_speed_card.dart';
import '../widgets/task_delete_dialog.dart';
import '../widgets/task_statistics/task_statistics_tab.dart';

class DownloadProgressWindowPage extends ConsumerStatefulWidget {
  const DownloadProgressWindowPage({super.key, required this.taskId, this.onClose, this.modal = false});

  final String taskId;
  final VoidCallback? onClose;
  final bool modal;

  @override
  ConsumerState<DownloadProgressWindowPage> createState() => _DownloadProgressWindowPageState();
}

class _DownloadProgressWindowPageState extends ConsumerState<DownloadProgressWindowPage> {
  bool _acting = false;
  bool _details = false;

  @override
  Widget build(BuildContext context) {
    final tasks = ref.watch(tasksControllerProvider);
    final task = tasks.value?.findTask(widget.taskId);
    final error = tasks.error ?? tasks.value?.lastError;
    final palette = AppPalette.of(context);
    final controller = ref.read(tasksControllerProvider.notifier);
    return Scaffold(
      child: SafeArea(
        child: Padding(
          padding: EdgeInsets.only(
            top: !widget.modal && AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (error != null)
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space16),
                  child: Text(
                    error.toString(),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: palette.error),
                  ),
                ),
              Expanded(
                child: task == null
                    ? Center(
                        child: tasks.isLoading
                            ? const CircularProgressIndicator()
                            : Text(error == null ? context.l10n.taskNotFound : context.l10n.failed),
                      )
                    : SingleChildScrollView(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            DownloadProgressHeader(task: task),
                            Padding(
                              padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space16),
                              child: DownloadSpeedCard(task: task),
                            ),
                            Padding(
                              padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space16),
                              child: GhostButton(
                                key: const ValueKey('download-progress-details'),
                                onPressed: () => setState(() => _details = !_details),
                                child: Row(
                                  children: [
                                    Expanded(child: Text(context.l10n.connectionStatus).small()),
                                    Icon(_details ? LucideIcons.chevronUp : LucideIcons.chevronDown, size: 16),
                                  ],
                                ),
                              ),
                            ),
                            if (_details) ...[
                              Padding(
                                padding: const EdgeInsets.all(AppDesignTokens.space16),
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.stretch,
                                  children: [
                                    SelectableText(task.url, key: const ValueKey('download-progress-url')).small(),
                                    const SizedBox(height: AppDesignTokens.space8),
                                    SelectableText(
                                      task.storagePath,
                                      key: const ValueKey('download-progress-path'),
                                    ).small(),
                                  ],
                                ),
                              ),
                              SizedBox(height: 240, child: TaskStatisticsTab(task: task, mobile: true)),
                            ],
                          ],
                        ),
                      ),
              ),
              const Divider(),
              DownloadProgressActions(
                task: task,
                busy: _acting,
                onClose: () => _runAction(_close),
                onCancel: () => _runAction(() => _cancel(task!)),
                onPause: () => _runAction(() => controller.pause(task!.id)),
                onResume: () => _runAction(() => controller.resume(task!.id)),
                onOpenFile: () => _openPath(task!.storagePath),
                onOpenDirectory: () => _openPath(task!.storagePath, reveal: true),
                onRetry: error == null ? null : () => _runAction(() => controller.refresh(silent: true)),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _cancel(TaskRecord task) async {
    final keepFiles = await showTaskDeleteDialog(
      context,
      taskCount: 1,
      keepFiles: true,
      memberCount: task.isGroup ? task.members.length : null,
      fileCount: task.isGroup ? task.files.length : null,
    );
    if (keepFiles == null || !mounted) return;
    await ref.read(tasksControllerProvider.notifier).deleteSelected([task.id], force: !keepFiles);
    await _close();
  }

  Future<void> _close() async {
    if (widget.onClose != null) {
      widget.onClose!();
    } else {
      await windowManager.close();
    }
  }

  Future<void> _openPath(String path, {bool reveal = false}) => _runAction(() async {
    final opened = reveal ? await FileExplorer.reveal(path) : await FileExplorer.open(path);
    if (!opened && mounted) {
      showAppToast(
        context,
        reveal ? context.l10n.unableLocatePath(path) : context.l10n.unableOpenPath(path),
        type: AppToastType.error,
      );
    }
  });

  Future<void> _runAction(Future<void> Function() action) async {
    if (_acting) return;
    setState(() => _acting = true);
    try {
      await action();
    } catch (error) {
      if (mounted) showAppToast(context, error.toString(), type: AppToastType.error);
    } finally {
      if (mounted) setState(() => _acting = false);
    }
  }
}
