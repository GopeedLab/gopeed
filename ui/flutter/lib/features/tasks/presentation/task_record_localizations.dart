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
    if (status == TaskStatus.completed) return null;
    if (remaining case final value?) return value;
    if (waiting) return l10n.waiting;
    if (status == TaskStatus.paused) return l10n.pause;
    final seconds = remainingSeconds;
    if (seconds == null) return null;
    return _formatTaskTime(l10n, seconds, remaining: true);
  }

  String? localizedDownloadDuration(AppLocalizations l10n) {
    final duration = downloadDuration;
    if (duration == null) return null;
    return formatTaskDuration(l10n, duration);
  }
}

String formatTaskDuration(AppLocalizations l10n, Duration duration) {
  final microseconds = duration.inMicroseconds;
  final seconds = microseconds <= 0
      ? 0
      : (microseconds + Duration.microsecondsPerSecond - 1) ~/ Duration.microsecondsPerSecond;
  return _formatTaskTime(l10n, seconds, remaining: false);
}

String _formatTaskTime(AppLocalizations l10n, int seconds, {required bool remaining}) {
  if (seconds < Duration.secondsPerMinute) {
    return remaining ? l10n.secondsRemaining(seconds) : l10n.secondsDuration(seconds);
  }
  if (seconds < Duration.secondsPerHour) {
    final minutes = (seconds / Duration.secondsPerMinute).ceil();
    return remaining ? l10n.minutesRemaining(minutes) : l10n.minutesDuration(minutes);
  }
  final hours = (seconds / Duration.secondsPerHour).ceil();
  return remaining ? l10n.hoursRemaining(hours) : l10n.hoursDuration(hours);
}
