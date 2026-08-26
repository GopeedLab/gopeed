import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../../shared/theme/app_design_tokens.dart';
import '../../../../../shared/theme/app_palette.dart';
import '../../../domain/task_record.dart';
import '../task_progress_bar.dart';
import 'task_card_parts.dart';

class TaskCardMobile extends StatefulWidget {
  const TaskCardMobile({
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
  State<TaskCardMobile> createState() => _TaskCardMobileState();
}

class _TaskCardMobileState extends State<TaskCardMobile> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final isDark = shad.Theme.of(context).brightness == Brightness.dark;
    final borderColor = isDark ? palette.border : palette.border.withValues(alpha: 0.6);
    final baseColor = isDark ? palette.cardBg : palette.cardBg.withValues(alpha: 0.92);
    final hoverColor = palette.taskCardHoverBg;
    final radius = BorderRadius.circular(AppDesignTokens.controlRadius);
    final task = widget.task;
    final fillColor = taskCardFillColor(palette, task.status, isDark);

    return MouseRegion(
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onPressed,
        onDoubleTap: !widget.batchMode && task.status == TaskStatus.completed ? widget.onOpen : null,
        child: Container(
          key: const ValueKey('task-card-mobile-surface'),
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
              children: [
                Positioned.fill(child: ColoredBox(color: baseColor)),
                if (_hovered)
                  Positioned.fill(
                    child: IgnorePointer(
                      child: ColoredBox(key: const ValueKey('task-card-hover-surface'), color: hoverColor),
                    ),
                  ),
                Positioned.fill(
                  child: IgnorePointer(
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        borderRadius: radius,
                        border: Border.all(color: _hovered ? palette.border : borderColor),
                      ),
                    ),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          AnimatedSize(
                            duration: const Duration(milliseconds: 160),
                            curve: Curves.easeOutCubic,
                            child: widget.batchMode
                                ? Padding(
                                    padding: const EdgeInsets.only(right: 10),
                                    child: shad.Checkbox(
                                      state: widget.selectedInBatch
                                          ? shad.CheckboxState.checked
                                          : shad.CheckboxState.unchecked,
                                      onChanged: (_) => widget.onToggleBatch(),
                                      borderColor: palette.border,
                                      backgroundColor: palette.cardBg,
                                      size: 18,
                                    ),
                                  )
                                : const SizedBox.shrink(),
                          ),
                          TaskCardIcon(task: task),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              task.name,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                color: palette.textPrimary,
                                fontSize: 14,
                                fontWeight: FontWeight.w500,
                                height: 1.1,
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          if (widget.listeningForUpdate) ...[
                            const TaskUpdateListeningIndicator(),
                            const SizedBox(width: 6),
                          ],
                          if (!widget.batchMode) ..._buildActionButtons(task, palette, isDark),
                        ],
                      ),
                      const SizedBox(height: 8),
                      if (task.status != TaskStatus.completed) ...[
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
                        TaskCardFooter(task: task, isError: task.status == TaskStatus.failed),
                      ] else ...[
                        TaskCardCompletedMeta(task: task),
                      ],
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

  List<Widget> _buildActionButtons(TaskRecord task, AppPalette palette, bool isDark) {
    final actions = switch (task.status) {
      TaskStatus.downloading => [(Icons.pause, widget.onPause), (Icons.close, widget.onDelete)],
      TaskStatus.paused => [(Icons.play_arrow, widget.onResume), (Icons.close, widget.onDelete)],
      TaskStatus.failed => [(Icons.refresh, widget.onResume), (Icons.delete_outline, widget.onDelete)],
      TaskStatus.completed => [(Icons.folder_open_outlined, widget.onReveal), (Icons.delete_outline, widget.onDelete)],
    };

    return actions
        .map(
          (action) => TaskCardActionButton(
            icon: action.$1,
            color: palette.taskCardAction,
            onPressed: action.$2 ?? () {},
            hoverBackground: isDark ? palette.border : Colors.transparent,
            buttonSize: 20,
            iconSize: 13,
          ),
        )
        .toList(growable: false);
  }
}
