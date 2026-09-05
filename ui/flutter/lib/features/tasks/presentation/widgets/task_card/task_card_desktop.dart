import 'package:flutter/gestures.dart' show TapUpDetails, kDoubleTapSlop, kDoubleTapTimeout;
import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../../shared/theme/app_design_tokens.dart';
import '../../../../../shared/theme/app_palette.dart';
import '../../../domain/task_record.dart';
import '../task_progress_bar.dart';
import 'task_card_parts.dart';

class TaskCardDesktop extends StatefulWidget {
  const TaskCardDesktop({
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
  final bool listeningForUpdate;

  @override
  State<TaskCardDesktop> createState() => _TaskCardDesktopState();
}

class _TaskCardDesktopState extends State<TaskCardDesktop> {
  bool _hovered = false;
  DateTime? _lastTapTime;
  Offset? _lastTapPosition;

  void _handleTap(TapUpDetails details) {
    final now = DateTime.now();
    final lastTapTime = _lastTapTime;
    final lastTapPosition = _lastTapPosition;
    final isDoubleTap =
        !widget.batchMode &&
        widget.task.status == TaskStatus.completed &&
        lastTapTime != null &&
        now.difference(lastTapTime) <= kDoubleTapTimeout &&
        lastTapPosition != null &&
        (details.globalPosition - lastTapPosition).distance <= kDoubleTapSlop;

    _lastTapTime = isDoubleTap ? null : now;
    _lastTapPosition = isDoubleTap ? null : details.globalPosition;
    widget.onPressed();
    if (isDoubleTap) {
      widget.onOpen?.call();
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final task = widget.task;
    final isDark = shad.Theme.of(context).brightness == Brightness.dark;
    final isError = task.status == TaskStatus.failed;
    final borderColor = isDark ? palette.border : palette.border.withValues(alpha: 0.6);
    final baseColor = isDark ? palette.cardBg : palette.cardBg.withValues(alpha: 0.92);
    final hoverColor = palette.taskCardHoverBg;
    final radius = BorderRadius.circular(AppDesignTokens.controlRadius);

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTapUp: _handleTap,
        child: Container(
          height: AppDesignTokens.taskRowHeight,
          decoration: BoxDecoration(
            borderRadius: radius,
            boxShadow: widget.selected
                ? [BoxShadow(color: palette.taskCardSelectedBorder, blurRadius: 0, spreadRadius: 1)]
                : null,
          ),
          child: ClipRRect(
            borderRadius: radius,
            clipBehavior: Clip.hardEdge,
            child: Stack(
              fit: StackFit.expand,
              children: [
                ColoredBox(color: baseColor),
                if (_hovered)
                  IgnorePointer(
                    child: ColoredBox(key: const ValueKey('task-card-hover-surface'), color: hoverColor),
                  ),
                IgnorePointer(
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      borderRadius: radius,
                      border: Border.all(color: _hovered ? palette.border : borderColor),
                    ),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 17, vertical: 13),
                  child: Row(
                    children: [
                      AnimatedContainer(
                        duration: const Duration(milliseconds: 220),
                        width: widget.batchMode ? 36 : 0,
                        child: widget.batchMode
                            ? Align(
                                alignment: Alignment.centerLeft,
                                child: shad.Checkbox(
                                  state: widget.selectedInBatch
                                      ? shad.CheckboxState.checked
                                      : shad.CheckboxState.unchecked,
                                  onChanged: (_) => widget.onToggleBatch(),
                                  borderColor: palette.border,
                                  backgroundColor: palette.cardBg,
                                  size: 20,
                                ),
                              )
                            : const SizedBox.shrink(),
                      ),
                      TaskCardIcon(task: task),
                      const SizedBox(width: 16),
                      Expanded(
                        child: SizedBox(
                          height: 46,
                          child: task.status == TaskStatus.completed
                              ? _CompletedContent(
                                  task: task,
                                  isDark: isDark,
                                  onReveal: widget.onReveal,
                                  onDelete: widget.onDelete,
                                  listeningForUpdate: widget.listeningForUpdate,
                                )
                              : _ProgressContent(
                                  task: task,
                                  isError: isError,
                                  isDark: isDark,
                                  onPause: widget.onPause,
                                  onResume: widget.onResume,
                                  onDelete: widget.onDelete,
                                  listeningForUpdate: widget.listeningForUpdate,
                                ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ProgressContent extends StatelessWidget {
  const _ProgressContent({
    required this.task,
    required this.isError,
    required this.isDark,
    this.onPause,
    this.onResume,
    this.onDelete,
    required this.listeningForUpdate,
  });

  final TaskRecord task;
  final bool isError;
  final bool isDark;
  final VoidCallback? onPause;
  final VoidCallback? onResume;
  final VoidCallback? onDelete;
  final bool listeningForUpdate;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final fillColor = taskCardFillColor(palette, task.status, isDark);

    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          height: 21,
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              Expanded(
                child: Text(
                  task.name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w500, height: 1),
                ),
              ),
              const SizedBox(width: 16),
              if (listeningForUpdate) ...[const TaskUpdateListeningIndicator(), const SizedBox(width: 8)],
              Row(mainAxisSize: MainAxisSize.min, children: _buildActionButtons(task, palette, isDark)),
            ],
          ),
        ),
        const SizedBox(height: 6),
        TaskProgressBar(
          value: task.progress,
          indeterminate: task.isIndeterminate,
          trackColor: palette.brandTrack,
          fillColor: fillColor,
          highlightStartColor: fillColor,
          highlightEndColor: fillColor,
          shimmer: task.status == TaskStatus.downloading,
        ),
        const SizedBox(height: 6),
        TaskCardFooter(task: task, isError: isError),
      ],
    );
  }

  List<Widget> _buildActionButtons(TaskRecord task, AppPalette palette, bool isDark) {
    final actions = switch (task.status) {
      TaskStatus.downloading => [(Icons.pause, onPause), (Icons.close, onDelete)],
      TaskStatus.paused => [(Icons.play_arrow, onResume), (Icons.close, onDelete)],
      TaskStatus.failed => [(Icons.refresh, onResume), (Icons.delete_outline, onDelete)],
      TaskStatus.completed => [(Icons.folder_open_outlined, null), (Icons.delete_outline, onDelete)],
    };

    return actions
        .map(
          (action) => TaskCardActionButton(
            icon: action.$1,
            color: palette.taskCardAction,
            onPressed: action.$2 ?? () {},
            hoverBackground: isDark ? palette.border : Colors.transparent,
          ),
        )
        .toList(growable: false);
  }
}

class _CompletedContent extends StatelessWidget {
  const _CompletedContent({
    required this.task,
    required this.isDark,
    required this.listeningForUpdate,
    this.onReveal,
    this.onDelete,
  });

  final TaskRecord task;
  final bool isDark;
  final VoidCallback? onReveal;
  final VoidCallback? onDelete;
  final bool listeningForUpdate;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);

    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          height: 21,
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              Expanded(
                child: Text(
                  task.name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: palette.textPrimary, fontSize: 14, fontWeight: FontWeight.w500, height: 1),
                ),
              ),
              const SizedBox(width: 16),
              if (listeningForUpdate) ...[const TaskUpdateListeningIndicator(), const SizedBox(width: 8)],
              Row(mainAxisSize: MainAxisSize.min, children: _buildActionButtons(task, palette, isDark)),
            ],
          ),
        ),
        const SizedBox(height: 6),
        TaskCardCompletedMeta(task: task),
      ],
    );
  }

  List<Widget> _buildActionButtons(TaskRecord task, AppPalette palette, bool isDark) {
    final actions = switch (task.status) {
      TaskStatus.downloading => [(Icons.pause, null), (Icons.close, onDelete)],
      TaskStatus.paused => [(Icons.play_arrow, null), (Icons.close, onDelete)],
      TaskStatus.failed => [(Icons.refresh, null), (Icons.delete_outline, onDelete)],
      TaskStatus.completed => [(Icons.folder_open_outlined, onReveal), (Icons.delete_outline, onDelete)],
    };

    return actions
        .map(
          (action) => TaskCardActionButton(
            icon: action.$1,
            color: palette.taskCardAction,
            onPressed: action.$2 ?? () {},
            hoverBackground: isDark ? palette.border : Colors.transparent,
          ),
        )
        .toList(growable: false);
  }
}
