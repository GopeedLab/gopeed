import 'package:flutter/widgets.dart';

import '../../../../l10n/l10n.dart';
import '../../domain/task_record.dart';
import '../task_record_localizations.dart';
import 'task_info_row.dart';

class TaskInfoSummary extends StatelessWidget {
  const TaskInfoSummary({super.key, required this.task});

  final TaskRecord task;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final completed = task.status == TaskStatus.completed;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TaskInfoRow(label: l10n.name, value: task.name),
        TaskInfoRow(label: l10n.status, value: task.localizedStatus(l10n), error: task.status == TaskStatus.failed),
        TaskInfoRow(
          label: l10n.size,
          value: completed
              ? task.total ?? task.downloaded
              : task.total != null
              ? '${task.downloaded} / ${task.total}'
              : task.downloaded,
        ),
        if (task.speed case final speed?) TaskInfoRow(label: l10n.speed, value: speed),
        if (task.uploading) TaskInfoRow(label: l10n.uploadSpeed, value: task.uploadSpeed!),
        if (completed)
          TaskInfoRow(label: l10n.downloadDuration, value: task.localizedDownloadDuration(l10n) ?? '—')
        else if (task.status == TaskStatus.paused)
          TaskInfoRow(label: l10n.remaining, value: '—')
        else if (task.localizedRemaining(l10n) case final remaining?)
          TaskInfoRow(label: l10n.remaining, value: remaining),
      ],
    );
  }
}
