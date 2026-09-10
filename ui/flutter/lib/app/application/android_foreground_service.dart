import 'dart:io';

import 'package:flutter_foreground_task/flutter_foreground_task.dart';

import '../../l10n/l10n.dart';

/// Keeps the Android process alive while Gopeed is available for downloads.
class AndroidForegroundService {
  const AndroidForegroundService._();

  static Future<void> ensureRunning(AppLocalizations l10n) async {
    if (!Platform.isAndroid) return;

    FlutterForegroundTask.init(
      androidNotificationOptions: AndroidNotificationOptions(
        channelId: 'gopeed_service',
        channelName: l10n.androidForegroundServiceChannel,
        channelImportance: NotificationChannelImportance.LOW,
        showWhen: true,
        priority: NotificationPriority.LOW,
      ),
      iosNotificationOptions: const IOSNotificationOptions(showNotification: false, playSound: false),
      foregroundTaskOptions: ForegroundTaskOptions(
        eventAction: ForegroundTaskEventAction.nothing(),
        autoRunOnBoot: true,
        allowWakeLock: true,
        allowWifiLock: true,
      ),
    );

    // Android 13+ requires runtime permission to show the service notification.
    // A denial only hides it from the notification drawer; the service can run.
    final permission = await FlutterForegroundTask.checkNotificationPermission();
    if (permission == NotificationPermission.denied) {
      await FlutterForegroundTask.requestNotificationPermission();
    }

    if (await FlutterForegroundTask.isRunningService) {
      _throwOnFailure(await FlutterForegroundTask.restartService());
      return;
    }

    _throwOnFailure(
      await FlutterForegroundTask.startService(
        notificationTitle: l10n.androidForegroundNotificationTitle,
        notificationText: l10n.androidForegroundNotificationText,
      ),
    );
  }

  static void _throwOnFailure(ServiceRequestResult result) {
    if (result case ServiceRequestFailure(:final error)) {
      throw error;
    }
  }
}
