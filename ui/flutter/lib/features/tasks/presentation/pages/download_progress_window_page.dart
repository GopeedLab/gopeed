import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../core/utils/file_explorer.dart';
import '../../../../core/window/app_window_bounds.dart';
import '../../../../core/window/app_window_chrome.dart';
import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/app_toast.dart';
import '../../application/tasks_controller.dart';
import '../../domain/task_record.dart';
import '../widgets/download_progress_actions.dart';
import '../widgets/download_progress_connections.dart';
import '../widgets/download_progress_header.dart';
import '../widgets/task_delete_dialog.dart';

class DownloadProgressWindowPage extends ConsumerStatefulWidget {
  const DownloadProgressWindowPage({super.key, required this.taskId});

  final String taskId;

  @override
  ConsumerState<DownloadProgressWindowPage> createState() => _DownloadProgressWindowPageState();
}

class _DownloadProgressWindowPageState extends ConsumerState<DownloadProgressWindowPage> {
  bool _acting = false;
  bool _detailsExpanded = false;
  bool _resizing = false;
  double? _compactHeight;

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
          padding: EdgeInsets.only(top: AppWindowChrome.reservesHeaderInset ? AppDesignTokens.windowHeaderHeight : 0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (error != null)
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: AppDesignTokens.space16,
                    vertical: AppDesignTokens.space4,
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          error.toString(),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: palette.error),
                        ),
                      ),
                      const SizedBox(width: AppDesignTokens.space8),
                      OutlineButton(
                        size: ButtonSize.small,
                        onPressed: _acting ? null : () => _runAction(() => controller.refresh(silent: true)),
                        child: Text(context.l10n.retry),
                      ),
                    ],
                  ),
                ),
              Expanded(
                child: task == null
                    ? Center(
                        child: tasks.isLoading
                            ? const CircularProgressIndicator()
                            : Text(error == null ? context.l10n.taskNotFound : context.l10n.failed),
                      )
                    : ScrollConfiguration(
                        behavior: ScrollConfiguration.of(context).copyWith(scrollbars: false),
                        child: CustomScrollView(
                          slivers: [
                            SliverPadding(
                              padding: const EdgeInsets.symmetric(
                                horizontal: AppDesignTokens.space16,
                                vertical: AppDesignTokens.space12,
                              ),
                              sliver: SliverToBoxAdapter(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.stretch,
                                  children: [
                                    DownloadProgressHeader(task: task),
                                    const SizedBox(height: AppDesignTokens.space8),
                                    DownloadProgressActions(
                                      task: task,
                                      busy: _acting,
                                      connectionsExpanded: _detailsExpanded,
                                      onToggleConnections: _resizing ? null : _toggleDetails,
                                      onCancel: () => _runAction(() => _cancel(task)),
                                      onPause: () => _runAction(() => controller.pause(task.id)),
                                      onResume: () => _runAction(() => controller.resume(task.id)),
                                      onOpenFile: () => _openPath(task.storagePath),
                                      onOpenDirectory: () => _openPath(task.storagePath, reveal: true),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                            if (_detailsExpanded && !task.isGroup)
                              SliverLayoutBuilder(
                                builder: (context, constraints) {
                                  final available =
                                      constraints.viewportMainAxisExtent -
                                      constraints.precedingScrollExtent -
                                      AppDesignTokens.space12;
                                  return SliverPadding(
                                    padding: const EdgeInsets.fromLTRB(
                                      AppDesignTokens.space16,
                                      0,
                                      AppDesignTokens.space16,
                                      AppDesignTokens.space12,
                                    ),
                                    sliver: SliverToBoxAdapter(
                                      child: SizedBox(
                                        height: available
                                            .clamp(96.0, AppDesignTokens.downloadProgressConnectionsHeight)
                                            .toDouble(),
                                        child: DownloadProgressConnections(task: task),
                                      ),
                                    ),
                                  );
                                },
                              ),
                          ],
                        ),
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _toggleDetails() async {
    if (_resizing) return;
    final expanded = !_detailsExpanded;
    setState(() {
      _detailsExpanded = expanded;
      _resizing = true;
      if (expanded) _compactHeight = null;
    });
    try {
      if (!AppWindowChrome.isDesktopWindow || await windowManager.isMaximized() || await windowManager.isFullScreen()) {
        return;
      }
      if (expanded) {
        final compact = await windowManager.getBounds();
        final height = await expandedWindowHeight(compact, AppDesignTokens.downloadProgressExpandedHeight);
        if (!mounted || compact.height == height) return;
        await windowManager.setSize(Size(compact.width, height));
        _compactHeight = compact.height;
      } else if (_compactHeight case final compactHeight?) {
        final current = await windowManager.getSize();
        if (!mounted) return;
        await windowManager.setSize(Size(current.width, compactHeight));
        _compactHeight = null;
      }
    } catch (error) {
      debugPrint('Unable to resize the download progress window: $error');
    } finally {
      if (mounted) setState(() => _resizing = false);
    }
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
    if (mounted) await windowManager.close();
  }

  Future<void> _openPath(String path, {bool reveal = false}) => _runAction(() async {
    final opened = reveal ? await FileExplorer.reveal(path) : await FileExplorer.open(path);
    if (!mounted) return;
    if (opened) {
      await windowManager.close();
    } else {
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
