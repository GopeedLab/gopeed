import 'package:flutter/material.dart' show Colors, Icons;
import 'package:flutter/widgets.dart';

import '../../../domain/task_record.dart';
import '../../../../../shared/theme/app_design_tokens.dart';
import '../../../../../shared/theme/app_palette.dart';
import '../../../../../shared/widgets/app_tooltip.dart';
import '../../../../../l10n/l10n.dart';
import '../../task_record_localizations.dart';

Color taskCardFillColor(AppPalette palette, TaskStatus status, bool isDark) {
  if (isDark) {
    switch (status) {
      case TaskStatus.failed:
        return palette.error;
      case TaskStatus.paused:
        return palette.textMuted;
      case TaskStatus.downloading:
      case TaskStatus.completed:
        return palette.brandProgress;
    }
  }

  switch (status) {
    case TaskStatus.failed:
      return palette.error;
    case TaskStatus.paused:
      return palette.textMuted;
    case TaskStatus.downloading:
    case TaskStatus.completed:
      return palette.brandProgress;
  }
}

class TaskUpdateListeningIndicator extends StatelessWidget {
  const TaskUpdateListeningIndicator({super.key});

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return AppTooltip(
      message: context.l10n.waitingForReplacementUrl,
      child: Icon(Icons.hearing, size: 15, color: palette.brandProgress),
    );
  }
}

class TaskCardIcon extends StatelessWidget {
  const TaskCardIcon({super.key, required this.task});

  final TaskRecord task;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);

    return SizedBox(
      width: 32,
      height: 32,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(AppDesignTokens.controlRadius),
        clipBehavior: Clip.hardEdge,
        child: ColoredBox(
          color: palette.taskCardIconBg,
          child: Center(child: Icon(task.icon, size: 20, color: palette.textSecondary)),
        ),
      ),
    );
  }
}

class TaskCardFooter extends StatelessWidget {
  const TaskCardFooter({super.key, required this.task, required this.isError});

  final TaskRecord task;
  final bool isError;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final metaColor = palette.taskMeta;
    final leftColor = isError ? palette.error : metaColor;
    final rightColor = isError ? palette.taskErrorMeta : metaColor;
    final statusText = switch (task.status) {
      TaskStatus.downloading => task.localizedRemaining(context.l10n) ?? context.l10n.downloading,
      TaskStatus.paused => task.localizedRemaining(context.l10n) ?? context.l10n.pause,
      TaskStatus.completed => context.l10n.completed,
      TaskStatus.failed => context.l10n.failed,
    };

    return SizedBox(
      height: 10,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Expanded(
            child: Text(
              task.total != null ? '${task.downloaded} / ${task.total}' : context.l10n.unknownSize,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: leftColor, fontSize: 10, height: 1),
            ),
          ),
          const SizedBox(width: 12),
          if (task.speed != null && !isError) ...[
            _TaskTransferSpeed(icon: Icons.south, value: task.speed!, color: metaColor),
            const SizedBox(width: 8),
            TickDivider(color: palette.progressTrack),
            const SizedBox(width: 8),
          ],
          if (task.uploading) ...[
            _TaskTransferSpeed(icon: Icons.north, value: task.uploadSpeed!, color: metaColor),
            const SizedBox(width: 8),
            TickDivider(color: palette.progressTrack),
            const SizedBox(width: 8),
          ],
          Text(
            statusText,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: rightColor, fontSize: 10, height: 1),
          ),
        ],
      ),
    );
  }
}

class TaskCardCompletedMeta extends StatelessWidget {
  const TaskCardCompletedMeta({super.key, required this.task});

  final TaskRecord task;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final metaColor = palette.taskMeta;

    return SizedBox(
      height: 10,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Expanded(
            child: Text(
              task.total ?? task.downloaded,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: metaColor, fontSize: 10, height: 1),
            ),
          ),
          const SizedBox(width: 12),
          if (task.speed != null) ...[
            _TaskTransferSpeed(icon: Icons.south, value: task.speed!, color: metaColor),
            const SizedBox(width: 8),
            TickDivider(color: palette.progressTrack),
            const SizedBox(width: 8),
          ],
          if (task.uploading) ...[
            _TaskTransferSpeed(icon: Icons.north, value: task.uploadSpeed!, color: metaColor),
            const SizedBox(width: 8),
            TickDivider(color: palette.progressTrack),
            const SizedBox(width: 8),
          ],
          Text(context.l10n.completed, style: TextStyle(color: palette.taskMetaSubtle, fontSize: 10, height: 1)),
        ],
      ),
    );
  }
}

class _TaskTransferSpeed extends StatelessWidget {
  const _TaskTransferSpeed({required this.icon, required this.value, required this.color});

  final IconData icon;
  final String value;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(icon, size: 10, color: color),
        const SizedBox(width: 2),
        Text(value, style: TextStyle(color: color, fontSize: 10, height: 1)),
      ],
    );
  }
}

class TickDivider extends StatelessWidget {
  const TickDivider({super.key, required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 4,
      height: 4,
      decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(999)),
    );
  }
}

class TaskCardActionButton extends StatelessWidget {
  const TaskCardActionButton({
    super.key,
    required this.icon,
    required this.color,
    required this.onPressed,
    required this.hoverBackground,
    this.buttonSize = 21,
    this.iconSize = 14,
  });

  final IconData icon;
  final Color color;
  final VoidCallback onPressed;
  final Color hoverBackground;
  final double buttonSize;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 4),
      child: HoverActionButton(
        hoverBackground: hoverBackground,
        onPressed: onPressed,
        child: SizedBox(
          width: buttonSize,
          height: buttonSize,
          child: Center(
            child: Icon(icon, size: iconSize, color: color),
          ),
        ),
      ),
    );
  }
}

class HoverActionButton extends StatefulWidget {
  const HoverActionButton({super.key, required this.child, required this.onPressed, required this.hoverBackground});

  final Widget child;
  final VoidCallback onPressed;
  final Color hoverBackground;

  @override
  State<HoverActionButton> createState() => _HoverActionButtonState();
}

class _HoverActionButtonState extends State<HoverActionButton> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: GestureDetector(
        onTap: widget.onPressed,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 140),
          decoration: BoxDecoration(
            color: _hovered ? widget.hoverBackground : Colors.transparent,
            borderRadius: BorderRadius.circular(4),
          ),
          child: widget.child,
        ),
      ),
    );
  }
}
