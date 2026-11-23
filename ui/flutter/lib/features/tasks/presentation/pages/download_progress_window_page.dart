import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform;
import 'package:shadcn_flutter/shadcn_flutter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:window_manager/window_manager.dart';

import '../../../../core/capabilities/app_capabilities.dart';
import '../../../../core/utils/file_explorer.dart';
import '../../../../core/window/app_window_bootstrap.dart';
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
  bool _closing = false;
  bool _detailsExpanded = false;
  bool _resizing = false;
  double _compactHeight = AppDesignTokens.downloadProgressWindowSize.height;
  var _targetHeights = (
    preferred: AppDesignTokens.downloadProgressWindowSize.height,
    minimum: AppDesignTokens.downloadProgressWindowSize.height,
  );

  @override
  void initState() {
    super.initState();
    ref.listenManual(tasksControllerProvider, (_, next) async {
      final snapshot = next.asData?.value;
      if (next.isLoading || next.hasError || snapshot == null || snapshot.lastError != null) return;
      if (snapshot.findTask(widget.taskId) == null) await _closeWindow();
    }, fireImmediately: true);
  }

  @override
  Widget build(BuildContext context) {
    final tasks = ref.watch(tasksControllerProvider);
    final task = tasks.value?.findTask(widget.taskId);
    final error = tasks.error ?? tasks.value?.lastError;
    final palette = AppPalette.of(context);
    final controller = ref.read(tasksControllerProvider.notifier);
    final content = Scaffold(
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
                child: task == null && error != null && !tasks.isLoading
                    ? Center(child: Text(context.l10n.failed))
                    : LayoutBuilder(
                        builder: (context, constraints) {
                          final expanded = task != null && _detailsExpanded && !task.isGroup;
                          final bodyHeight = constraints.maxHeight;
                          final inset = MediaQuery.sizeOf(context).height - bodyHeight;
                          final summaryHeight = expanded ? (_compactHeight - inset).clamp(0.0, bodyHeight) : bodyHeight;
                          return Padding(
                            padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space16),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                SizedBox(
                                  height: summaryHeight,
                                  child: Padding(
                                    padding: const EdgeInsets.symmetric(vertical: AppDesignTokens.space8),
                                    child: Column(
                                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                      crossAxisAlignment: CrossAxisAlignment.stretch,
                                      children: [
                                        Flexible(
                                          child: DownloadProgressHeader(
                                            task: task,
                                            onHeightDelta: (delta) {
                                              if (task == null || !mounted || _closing) return;
                                              final compact = math.max(
                                                AppDesignTokens.downloadProgressWindowSize.height,
                                                (inset + summaryHeight + delta).ceilToDouble(),
                                              );
                                              if (compact != _compactHeight) setState(() => _compactHeight = compact);
                                              _fitWindow(expanded);
                                            },
                                          ),
                                        ),
                                        const SizedBox(height: AppDesignTokens.space12),
                                        if (task != null)
                                          DownloadProgressActions(
                                            task: task,
                                            busy: _acting,
                                            connectionsExpanded: _detailsExpanded,
                                            onToggleConnections: () =>
                                                setState(() => _detailsExpanded = !_detailsExpanded),
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
                                if (expanded)
                                  SizedBox(
                                    height: math.max(0.0, bodyHeight - summaryHeight - AppDesignTokens.space8),
                                    child: DownloadProgressConnections(task: task),
                                  ),
                              ],
                            ),
                          );
                        },
                      ),
              ),
            ],
          ),
        ),
      ),
    );
    final media = MediaQuery.of(context);
    return MediaQuery(
      data: media.copyWith(textScaler: _ProgressTextScaler(media.textScaler)),
      child: ScrollConfiguration(behavior: ScrollConfiguration.of(context).copyWith(scrollbars: false), child: content),
    );
  }

  void _fitWindow(bool expanded) {
    if (!AppWindowChrome.isDesktopWindow || _closing) return;
    final heights = (
      preferred: _compactHeight + (expanded ? AppDesignTokens.downloadProgressDefaultDetailsHeight : 0.0),
      minimum: _compactHeight + (expanded ? AppDesignTokens.downloadProgressMinimumDetailsHeight : 0.0),
    );
    if (heights == _targetHeights) return;
    _targetHeights = heights;
    if (!_resizing) unawaited(_resizeWindow());
  }

  Future<void> _resizeWindow() async {
    _resizing = true;
    ({double preferred, double minimum}) target;
    do {
      target = _targetHeights;
      try {
        final platform = defaultTargetPlatform;
        // AppKit can report non-resizable windows as zoomed even at their compact size.
        if ((platform != TargetPlatform.macOS && await windowManager.isMaximized()) ||
            await windowManager.isFullScreen()) {
          break;
        }
        final current = await windowManager.getBounds();
        final next = await fitWindowHeight(
          current,
          preferredHeight: target.preferred,
          minimumHeight: target.minimum,
          windowScale: platform == TargetPlatform.windows ? windowManager.getDevicePixelRatio() : null,
        );
        if (mounted && !_closing && target == _targetHeights) {
          final move = next.topLeft != current.topLeft;
          if (platform == TargetPlatform.linux) {
            if (move) await windowManager.setPosition(next.topLeft);
            // A previous GTK size request may still be waiting for allocation.
            await AppWindowBootstrap.setLinuxProgressWindowSize(next.size);
          } else if (move) {
            await windowManager.setBounds(next);
          } else if (next.height != current.height) {
            await windowManager.setSize(next.size);
          }
        }
      } catch (error) {
        debugPrint('Unable to resize the download progress window: $error');
      }
    } while (mounted && !_closing && target != _targetHeights);
    _resizing = false;
  }

  Future<void> _cancel(TaskRecord task) async {
    final keepFiles = await showRememberedTaskDeleteDialog(
      context,
      storage: ref.read(appStorageServiceProvider),
      taskCount: 1,
      memberCount: task.isGroup ? task.members.length : null,
      fileCount: task.isGroup ? task.files.length : null,
    );
    if (keepFiles == null || !mounted) return;
    await ref.read(tasksControllerProvider.notifier).deleteSelected([task.id], force: !keepFiles);
    if (mounted) await _closeWindow();
  }

  Future<void> _closeWindow() async {
    if (_closing) return;
    _closing = true;
    try {
      await windowManager.close();
    } catch (error) {
      _closing = false;
      if (mounted) showAppToast(context, error.toString(), type: AppToastType.error);
    }
  }

  Future<void> _openPath(String path, {bool reveal = false}) => _runAction(() async {
    final opened = reveal ? await FileExplorer.reveal(path) : await FileExplorer.open(path);
    if (!mounted) return;
    if (opened) {
      await _closeWindow();
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

/// Use the desktop menu size as the minimum while preserving system text scaling.
class _ProgressTextScaler extends TextScaler {
  const _ProgressTextScaler(this.system);
  final TextScaler system;

  @override
  double scale(double fontSize) =>
      fontSize == 0 ? 0 : system.scale(math.max(fontSize, AppDesignTokens.desktopMenuFontSize));

  @override
  double get textScaleFactor => system.scale(AppDesignTokens.desktopMenuFontSize) / AppDesignTokens.desktopMenuFontSize;
}
