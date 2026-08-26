import '../../../l10n/app_localizations.dart';
import '../domain/task_record.dart';

extension TaskRecordLocalizations on TaskRecord {
  String localizedStatus(AppLocalizations l10n) => switch (status) {
    TaskStatus.downloading when waiting => l10n.waiting,
    TaskStatus.downloading => l10n.downloading,
    TaskStatus.completed => l10n.completed,
    TaskStatus.paused => l10n.pause,
    TaskStatus.failed => l10n.failed,
  };

  String localizedStatusUppercase(AppLocalizations l10n) => localizedStatus(l10n).toUpperCase();

  String? localizedRemaining(AppLocalizations l10n) {
    if (remaining case final value?) return value;
    if (waiting) return l10n.waiting;
    if (status == TaskStatus.paused) return l10n.pause;
    if (status == TaskStatus.completed) return l10n.completed;
    final seconds = remainingSeconds;
    if (seconds == null) return null;
    if (seconds < 60) return l10n.secondsRemaining(seconds);
    if (seconds < 3600) return l10n.minutesRemaining((seconds / 60).ceil());
    return l10n.hoursRemaining((seconds / 3600).ceil());
  }

  String localizedError(AppLocalizations l10n) => error ?? l10n.taskFailed;

  String localizedCompletedLabel(AppLocalizations l10n) => completedLabel ?? l10n.downloaded;
}
