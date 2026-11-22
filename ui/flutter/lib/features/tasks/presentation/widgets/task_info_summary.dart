import 'package:flutter/widgets.dart';

import '../../../../l10n/l10n.dart';
import '../../../../shared/theme/app_design_tokens.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';
import 'task_info_row.dart';

class TaskInfoSummary extends StatelessWidget {
  const TaskInfoSummary({super.key, required this.task, this.compact = false});

  final TaskRecord? task;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final task = this.task;
    final completed = task?.status == TaskStatus.completed;
    final remaining = task?.localizedRemaining(l10n);
    var size = task?.downloaded;
    if (task?.total case final total?) {
      size = completed ? total : '$size / $total';
    }
    final spacing = compact ? AppDesignTokens.space8 : 14.0;
    Widget row({required String label, required String? value, bool error = false}) =>
        TaskInfoRow(label: label, value: value ?? '', error: error, bottomPadding: spacing);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TaskInfoRow(label: l10n.name, value: task?.name ?? '', bottomPadding: spacing, scrollValue: compact),
        row(label: l10n.status, value: task?.localizedStatus(l10n), error: task?.status == TaskStatus.failed),
        row(label: l10n.size, value: size),
        if (compact || task?.speed != null) row(label: l10n.speed, value: task?.speed),
        if (task?.uploading == true) row(label: l10n.uploadSpeed, value: task?.uploadSpeed),
        if (completed)
          row(label: l10n.downloadDuration, value: task?.localizedDownloadDuration(l10n) ?? (compact ? '' : '—'))
        else if (task?.status == TaskStatus.paused)
          row(label: l10n.remaining, value: compact ? '' : '—')
        else if (compact || remaining != null)
          row(label: l10n.remaining, value: compact && task?.waiting == true ? null : remaining),
      ],
    );
  }
}
