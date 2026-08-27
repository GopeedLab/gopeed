import 'package:flutter/material.dart' show Icons;
import 'package:flutter/widgets.dart';
import 'package:shadcn_flutter/shadcn_flutter.dart' as shad;

import '../../../../api/model/task.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../../../shared/widgets/file_tree_view.dart';
import '../../../../util/util.dart';
import '../../../../l10n/l10n.dart';
import '../../domain/task_record.dart';
import 'task_progress_bar.dart';

class TaskFileTree extends StatefulWidget {
  const TaskFileTree({super.key, required this.task, this.runtimeStatus, this.progressAction});

  final TaskRecord task;
  final TaskRuntimeStatus? runtimeStatus;
  final Widget? progressAction;

  @override
  State<TaskFileTree> createState() => _TaskFileTreeState();
}

class _TaskFileTreeState extends State<TaskFileTree> {
  late List<FileTreeItem<TaskFileNode>> _items;

  @override
  void initState() {
    super.initState();
    _items = _buildItems(widget.task.files);
  }

  @override
  void didUpdateWidget(covariant TaskFileTree oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.task.id != oldWidget.task.id || !identical(widget.task.files, oldWidget.task.files)) {
      _items = _buildItems(widget.task.files);
    }
  }

  @override
  Widget build(BuildContext context) {
    final progressByIndex = {
      for (final file in widget.runtimeStatus?.files ?? const <FileRuntimeStatus>[]) file.index: file,
    };

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _TaskDownloadProgress(task: widget.task, runtimeStatus: widget.runtimeStatus, action: widget.progressAction),
        const SizedBox(height: 12),
        Expanded(
          child: FileTreeView<TaskFileNode>(
            items: _items,
            keyPrefix: 'task-file-tree',
            rowHeight: 27,
            iconSize: 15,
            iconBuilder: (node) => taskFileTypeIcon(node.label, isFolder: node.isFolder),
            contentTextStyle: const TextStyle(fontSize: 11, height: 1.1),
            trailingHeader: const _TaskFileColumnsHeader(),
            trailingBuilder: (context, node) => _FileDownloadProgress(
              node: node,
              progressByIndex: progressByIndex,
              runtimeStatusAvailable: widget.runtimeStatus != null,
              fallbackProgress: widget.task.progress,
              taskStatus: widget.task.status,
            ),
          ),
        ),
      ],
    );
  }

  List<FileTreeItem<TaskFileNode>> _buildItems(List<TaskFileNode> files) {
    return files
        .asMap()
        .entries
        .map(
          (entry) => FileTreeItem<TaskFileNode>(
            key: entry.key.toString(),
            path: entry.value.path,
            name: entry.value.name,
            size: entry.value.sizeBytes,
            data: entry.value,
          ),
        )
        .toList(growable: false);
  }
}

class _TaskDownloadProgress extends StatelessWidget {
  const _TaskDownloadProgress({required this.task, required this.runtimeStatus, this.action});

  final TaskRecord task;
  final TaskRuntimeStatus? runtimeStatus;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final runtime = runtimeStatus;
    final totalBytes = runtime?.total ?? 0;
    final downloadedBytes = runtime?.downloaded ?? 0;
    final progress = runtime == null
        ? task.progress
        : totalBytes > 0
        ? (downloadedBytes / totalBytes).clamp(0.0, 1.0)
        : null;
    final progressLabel = progress == null ? '--' : '${(progress * 100).toStringAsFixed(progress < 0.1 ? 1 : 0)}%';
    final transferred = runtime == null
        ? (task.total == null ? task.downloaded : '${task.downloaded} / ${task.total}')
        : totalBytes > 0
        ? '${Util.fmtByte(downloadedBytes)} / ${Util.fmtByte(totalBytes)}'
        : Util.fmtByte(downloadedBytes);

    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      context.l10n.taskProgress,
                      style: TextStyle(color: palette.textPrimary, fontSize: 12, fontWeight: FontWeight.w600),
                    ),
                  ),
                  Text(transferred, style: TextStyle(color: palette.textSecondary, fontSize: 11)),
                  const SizedBox(width: 8),
                  SizedBox(
                    width: 38,
                    child: Text(
                      progressLabel,
                      textAlign: TextAlign.right,
                      style: TextStyle(color: palette.textPrimary, fontSize: 11, fontWeight: FontWeight.w600),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 7),
              TaskProgressBar(
                key: const ValueKey('task-files-total-progress'),
                value: progress,
                indeterminate: runtime == null && task.isIndeterminate,
                shimmer: task.status == TaskStatus.downloading,
                height: 4,
                trackColor: palette.progressTrack,
                fillColor: palette.brandProgress,
                highlightStartColor: palette.brandProgress.withValues(alpha: 0),
                highlightEndColor: palette.brandProgress.withValues(alpha: 0),
              ),
            ],
          ),
        ),
        if (action != null) ...[const SizedBox(width: 12), action!],
      ],
    );
  }
}

class _FileDownloadProgress extends StatelessWidget {
  const _FileDownloadProgress({
    required this.node,
    required this.progressByIndex,
    required this.runtimeStatusAvailable,
    required this.fallbackProgress,
    required this.taskStatus,
  });

  final FileTreeNode<TaskFileNode> node;
  final Map<int, FileRuntimeStatus> progressByIndex;
  final bool runtimeStatusAvailable;
  final double? fallbackProgress;
  final TaskStatus taskStatus;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final taskStateIcon = switch (taskStatus) {
      TaskStatus.paused => Semantics(
        label: context.l10n.pause,
        child: Icon(Icons.pause_circle_filled, size: 16, color: palette.textMuted),
      ),
      TaskStatus.failed => Semantics(
        label: context.l10n.failed,
        child: Icon(Icons.error_rounded, size: 16, color: palette.error),
      ),
      TaskStatus.completed => Semantics(
        label: context.l10n.completed,
        child: Icon(shad.BootstrapIcons.checkCircleFill, size: 16, color: palette.brandProgress),
      ),
      TaskStatus.downloading => null,
    };
    if (taskStateIcon != null) {
      return SizedBox(
        key: ValueKey('task-file-progress-${node.key}'),
        width: 116,
        child: _TaskFileColumns(progress: taskStateIcon, size: _formatFileSize(node.size)),
      );
    }

    final selectedFiles = node.leafItems
        .map((item) => int.tryParse(item.key))
        .whereType<int>()
        .map((index) => progressByIndex[index])
        .whereType<FileRuntimeStatus>()
        .toList(growable: false);
    if (selectedFiles.isEmpty) {
      final progress = runtimeStatusAvailable ? null : fallbackProgress;
      final showFallbackProgress = !runtimeStatusAvailable && progress != null;
      return SizedBox(
        key: showFallbackProgress ? ValueKey('task-file-progress-${node.key}') : null,
        width: 116,
        child: _TaskFileColumns(
          progress: showFallbackProgress
              ? Semantics(
                  label: '${(progress * 100).toStringAsFixed(0)}%',
                  child: shad.CircularProgressIndicator(
                    value: progress,
                    size: 16,
                    strokeWidth: 2,
                    color: palette.brandProgress,
                    backgroundColor: palette.progressTrack,
                  ),
                )
              : const SizedBox.square(dimension: 16),
          size: _formatFileSize(node.size),
        ),
      );
    }

    final downloaded = selectedFiles.fold<int>(0, (total, file) => total + file.downloaded);
    final sizeKnown = selectedFiles.every((file) => file.size > 0);
    final size = sizeKnown ? selectedFiles.fold<int>(0, (total, file) => total + file.size) : 0;
    final progress = size > 0 ? (downloaded / size).clamp(0.0, 1.0) : null;
    final completed = progress != null && progress >= 1;
    final progressLabel = progress == null ? Util.fmtByte(downloaded) : '${(progress * 100).toStringAsFixed(0)}%';

    return SizedBox(
      key: ValueKey('task-file-progress-${node.key}'),
      width: 116,
      child: _TaskFileColumns(
        progress: Semantics(
          label: completed ? context.l10n.completed : progressLabel,
          child: completed
              ? Icon(shad.BootstrapIcons.checkCircleFill, size: 16, color: palette.brandProgress)
              : shad.CircularProgressIndicator(
                  value: progress ?? (downloaded > 0 ? null : 0),
                  size: 16,
                  strokeWidth: 2,
                  color: palette.brandProgress,
                  backgroundColor: palette.progressTrack,
                ),
        ),
        size: _formatFileSize(node.size),
      ),
    );
  }
}

class _TaskFileColumnsHeader extends StatelessWidget {
  const _TaskFileColumnsHeader();

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final style = TextStyle(color: palette.textSecondary, fontSize: 10, fontWeight: FontWeight.w600);
    return _TaskFileColumns(
      progress: Text(context.l10n.progress, maxLines: 1, style: style),
      size: context.l10n.size,
      sizeStyle: style,
    );
  }
}

class _TaskFileColumns extends StatelessWidget {
  const _TaskFileColumns({required this.progress, required this.size, this.sizeStyle});

  final Widget progress;
  final String size;
  final TextStyle? sizeStyle;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return Row(
      children: [
        SizedBox(width: 44, child: Center(child: progress)),
        const SizedBox(width: 4),
        Expanded(
          child: Text(
            size,
            maxLines: 1,
            overflow: TextOverflow.fade,
            softWrap: false,
            textAlign: TextAlign.right,
            style: sizeStyle ?? TextStyle(color: palette.textSecondary, fontSize: 10),
          ),
        ),
      ],
    );
  }
}

String _formatFileSize(int size) => size > 0 ? Util.fmtByte(size) : '--';
