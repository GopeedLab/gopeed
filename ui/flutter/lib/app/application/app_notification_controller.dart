import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as path;

import '../../core/common/task_event.dart';
import '../../core/libgopeed_boot.dart';
import '../../l10n/l10n.dart';
import '../../util/log_util.dart';
import '../../util/util.dart';
import 'app_runtime_controller.dart';

final appNotificationControllerProvider = AsyncNotifierProvider<AppNotificationController, AppNotificationState>(
  AppNotificationController.new,
);

class AppNotificationState {
  const AppNotificationState({this.started = false});

  final bool started;
}

class AppNotificationController extends AsyncNotifier<AppNotificationState> {
  final FlutterLocalNotificationsPlugin _plugin = FlutterLocalNotificationsPlugin();
  StreamSubscription<TaskEvent>? _taskEventSubscription;
  var _notificationId = 0;

  @visibleForTesting
  Stream<TaskEvent> get taskEvents => LibgopeedBoot.instance.taskEvents;

  @override
  Future<AppNotificationState> build() async {
    final runtime = ref.watch(appRuntimeControllerProvider).value;
    if (runtime == null || kIsWeb || !Util.isDesktop()) {
      return const AppNotificationState();
    }
    await _initNotifications(
      appLocalizationsFor(runtime.downloaderConfig.extra.locale),
      requestPermissions: runtime.downloaderConfig.extra.desktopNotification,
    );
    _listenTaskEvents();
    ref.onDispose(() {
      unawaited(_taskEventSubscription?.cancel());
    });
    return const AppNotificationState(started: true);
  }

  Future<void> _initNotifications(AppLocalizations locale, {required bool requestPermissions}) async {
    // The runtime is watched, so enabling notifications in settings also
    // requests authorization. macOS remembers previously granted/denied access.
    final darwin = DarwinInitializationSettings(
      requestAlertPermission: requestPermissions,
      requestBadgePermission: false,
      requestSoundPermission: requestPermissions,
    );
    final linux = LinuxInitializationSettings(
      defaultActionName: locale.open,
      defaultIcon: AssetsLinuxIcon('assets/icon/icon.png'),
    );

    String? windowsIconPath;
    try {
      if (Util.isWindows()) {
        // Windows requires a file path for IconUri, so use the bundled PNG
        // beside the executable rather than copying it to another directory.
        final file = File(
          path.join(
            path.dirname(Platform.resolvedExecutable),
            'data',
            'flutter_assets',
            'assets',
            'icon',
            'icon_512.png',
          ),
        );
        if (await file.exists()) {
          windowsIconPath = file.absolute.path;
        } else {
          logger.w('Windows notification icon not found: ${file.path}');
        }
      }
    } catch (error, stackTrace) {
      logger.w('prepare Windows notification icon failed', error, stackTrace);
    }

    final windows = WindowsInitializationSettings(
      appName: 'Gopeed',
      appUserModelId: 'com.gopeed.gopeed',
      guid: '3c1bf3f4-3d91-4eaa-a33f-8705e71cf1ce',
      iconPath: windowsIconPath,
    );

    await _plugin.initialize(
      settings: InitializationSettings(macOS: darwin, linux: linux, windows: windows),
    );
  }

  void _listenTaskEvents() {
    _taskEventSubscription?.cancel();
    _taskEventSubscription = taskEvents.listen((event) async {
      final runtime = ref.read(appRuntimeControllerProvider).value;
      if (runtime?.downloaderConfig.extra.desktopNotification == false) return;
      final locale = appLocalizationsFor(runtime?.downloaderConfig.extra.locale ?? '');
      switch (event.type) {
        case TaskEventType.done:
          await _showNotification(title: locale.notificationTaskDone, body: event.name);
        case TaskEventType.error:
          await _showNotification(title: locale.notificationTaskError, body: event.name);
      }
    });
  }

  Future<void> _showNotification({required String title, required String body}) async {
    const details = NotificationDetails(
      macOS: DarwinNotificationDetails(),
      linux: LinuxNotificationDetails(),
      windows: WindowsNotificationDetails(),
    );
    await _plugin.show(id: _notificationId++, title: title, body: body, notificationDetails: details);
  }
}
