import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/features/tasks/domain/task_record.dart';
import 'package:gopeed/features/tasks/presentation/task_record_localizations.dart';
import 'package:gopeed/l10n/app_localizations_en.dart';

void main() {
  test('remaining time and download duration share unit thresholds and rounding', () {
    final l10n = AppLocalizationsEn();
    final cases = <(int, String, String)>[
      (59, '59 seconds remaining', '59 seconds'),
      (60, '1 minute remaining', '1 minute'),
      (61, '2 minutes remaining', '2 minutes'),
      (3599, '60 minutes remaining', '60 minutes'),
      (3600, '1 hour remaining', '1 hour'),
      (3723, '2 hours remaining', '2 hours'),
    ];

    for (final (seconds, remaining, duration) in cases) {
      final runningTask = TaskRecord(
        id: 'running-$seconds',
        name: 'archive.zip',
        status: TaskStatus.downloading,
        downloaded: '1 MB',
        url: 'https://example.com/archive.zip',
        storagePath: '/downloads/archive.zip',
        files: const [],
        uploading: false,
        remainingSeconds: seconds,
      );
      final completedTask = TaskRecord(
        id: 'completed-$seconds',
        name: 'archive.zip',
        status: TaskStatus.completed,
        downloaded: '1 MB',
        url: 'https://example.com/archive.zip',
        storagePath: '/downloads/archive.zip',
        files: const [],
        uploading: false,
        downloadDuration: Duration(seconds: seconds),
      );

      expect(runningTask.localizedRemaining(l10n), remaining);
      expect(completedTask.localizedDownloadDuration(l10n), duration);
      expect(formatTaskDuration(l10n, Duration(seconds: seconds)), duration);
    }
  });
}
