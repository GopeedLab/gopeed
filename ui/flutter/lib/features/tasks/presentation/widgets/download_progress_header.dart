import 'package:shadcn_flutter/shadcn_flutter.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../../../shared/theme/app_palette.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';
import 'task_card/task_card_parts.dart';
import 'task_info_summary.dart';

class DownloadProgressHeader extends StatelessWidget {
  const DownloadProgressHeader({super.key, required this.task});

  final TaskRecord task;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final progress = task.status == TaskStatus.completed ? 1.0 : task.progress;
    final percentage = progress == null ? null : '${(progress * 100).toStringAsFixed(1)}%';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TaskInfoSummary(task: task),
        if (task.error case final error? when error.isNotEmpty) ...[
          Text(
            error,
            maxLines: 3,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: AppPalette.of(context).error, fontSize: 12),
          ),
          const SizedBox(height: AppDesignTokens.space8),
        ],
        Semantics(
          label: l10n.progress,
          value: percentage ?? task.localizedStatus(l10n),
          child: TaskCardProgressBar(task: task, height: 3),
        ),
      ],
    );
  }
}
