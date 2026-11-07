import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';
import 'task_progress_bar.dart';

class DownloadProgressHeader extends StatelessWidget {
  const DownloadProgressHeader({super.key, required this.task});

  final TaskRecord task;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final progress = task.status == TaskStatus.completed ? 1.0 : task.progress;
    final color = task.status == TaskStatus.failed ? palette.error : palette.taskCardProgressFill;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: AppDesignTokens.space16, vertical: AppDesignTokens.space8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(AppDesignTokens.space8),
                decoration: BoxDecoration(
                  color: palette.brandTrack,
                  borderRadius: BorderRadius.circular(AppDesignTokens.space8),
                ),
                child: Icon(
                  task.isFolder || task.isGroup ? Icons.folder_outlined : Icons.insert_drive_file_outlined,
                  size: 20,
                  color: palette.textSecondary,
                ),
              ),
              const SizedBox(width: AppDesignTokens.space12),
              Expanded(
                flex: 4,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(task.name, maxLines: 1, overflow: TextOverflow.ellipsis).small().semiBold(),
                    Text(
                      [
                        if (task.protocol != null) task.protocol!.name.toUpperCase(),
                        if (Uri.tryParse(task.url)?.host.isNotEmpty == true) Uri.parse(task.url).host,
                      ].join(' · '),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ).xSmall().muted(),
                  ],
                ),
              ),
              const SizedBox(width: AppDesignTokens.space8),
              Flexible(
                child: Align(
                  alignment: AlignmentDirectional.centerEnd,
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: AppDesignTokens.space8,
                      vertical: AppDesignTokens.space4,
                    ),
                    decoration: BoxDecoration(
                      color: palette.brandTrack,
                      borderRadius: BorderRadius.circular(AppDesignTokens.windowRadius),
                    ),
                    child: Text(
                      task.localizedStatus(context.l10n),
                      textAlign: TextAlign.end,
                      style: TextStyle(color: task.status == TaskStatus.failed ? palette.error : palette.textPrimary),
                    ).xSmall(),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: AppDesignTokens.space8),
          Row(
            children: [
              Expanded(child: Text('${task.downloaded} / ${task.total ?? context.l10n.unknownSize}').xSmall()),
              if (progress != null) Text('${(progress * 100).toStringAsFixed(1)}%').large().semiBold(),
            ],
          ),
          const SizedBox(height: AppDesignTokens.space8),
          Semantics(
            label: context.l10n.progress,
            value: progress == null ? task.localizedStatus(context.l10n) : '${(progress * 100).toStringAsFixed(1)}%',
            child: TaskProgressBar(
              value: progress,
              indeterminate: task.isIndeterminate,
              shimmer: task.status == TaskStatus.downloading && !task.waiting,
              height: AppDesignTokens.space8,
              trackColor: palette.brandTrack,
              fillColor: color,
              highlightStartColor: color,
              highlightEndColor: color,
            ),
          ),
          const SizedBox(height: AppDesignTokens.space8),
          if (task.error?.isNotEmpty == true)
            Text(
              task.error!,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: palette.error),
            ),
        ],
      ),
    );
  }
}
